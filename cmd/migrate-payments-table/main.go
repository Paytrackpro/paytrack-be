package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"code.cryptopower.dev/mgmt-ng/be/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OldPaymentSetting represents the current structure without network field
type OldPaymentSetting struct {
	Type    utils.Method `json:"type"`
	Address string       `json:"address"`
}

// NewPaymentSetting represents the new structure with network field
type NewPaymentSetting struct {
	Coin    string `json:"coin"`
	Network string `json:"network"`
	Address string `json:"address"`
}

// MigratePaymentsSettings migrates payment_settings in payments table to new format
func MigratePaymentsSettings() {
	// Get database connection string from environment variable
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	// Connect to database
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Start migration
	log.Println("Starting payment_settings migration in payments table...")
	
	// Begin transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Fatal("Migration failed with panic:", r)
		}
	}()

	// First, get all payment IDs and settings into memory to avoid connection issues
	type PaymentData struct {
		ID       uint64
		Settings []byte
	}
	
	var payments []PaymentData
	rows, err := tx.Raw(`
		SELECT id, payment_settings 
		FROM payments 
		WHERE payment_settings IS NOT NULL 
		AND payment_settings::text != 'null' 
		AND payment_settings::text != '[]'
	`).Rows()
	if err != nil {
		tx.Rollback()
		log.Fatal("Failed to query payments:", err)
	}
	
	// Load all data into memory first
	for rows.Next() {
		var payment PaymentData
		if err := rows.Scan(&payment.ID, &payment.Settings); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}
		payments = append(payments, payment)
	}
	rows.Close() // Close rows immediately after reading
	
	log.Printf("Found %d payments with payment_settings to process", len(payments))

	migratedCount := 0
	skippedCount := 0
	errorCount := 0

	// Now process each payment
	for _, payment := range payments {
		paymentId := payment.ID
		paymentSettingsJSON := payment.Settings

		// Parse old payment settings
		var oldSettings []OldPaymentSetting
		if err := json.Unmarshal(paymentSettingsJSON, &oldSettings); err != nil {
			// Try to parse as new format (might already be migrated)
			var testNewSettings []NewPaymentSetting
			if err2 := json.Unmarshal(paymentSettingsJSON, &testNewSettings); err2 == nil {
				// Already in new format, skip
				log.Printf("Payment %d already migrated, skipping", paymentId)
				skippedCount++
				continue
			}
			
			log.Printf("Error parsing payment_settings for payment %d: %v", paymentId, err)
			errorCount++
			continue
		}

		// Convert to new format
		newSettings := make([]NewPaymentSetting, 0, len(oldSettings))
		
		for _, oldSetting := range oldSettings {
			coin, network := mapMethodToCoinNetwork(oldSetting.Type)
			
			if coin == "" || network == "" {
				log.Printf("Warning: Unknown payment type %d for payment %d, using fallback", 
					oldSetting.Type, paymentId)
				// Use fallback for unknown types
				coin = fmt.Sprintf("UNKNOWN_%d", oldSetting.Type)
				network = "unknown"
			}
			
			newSettings = append(newSettings, NewPaymentSetting{
				Coin:    coin,
				Network: network,
				Address: oldSetting.Address,
			})
		}

		// Convert new settings to JSON
		newSettingsJSON, err := json.Marshal(newSettings)
		if err != nil {
			log.Printf("Error marshaling new settings for payment %d: %v", paymentId, err)
			errorCount++
			continue
		}

		// Update payment_settings column
		if err := tx.Exec("UPDATE payments SET payment_settings = ? WHERE id = ?", 
			newSettingsJSON, paymentId).Error; err != nil {
			log.Printf("Error updating payment %d: %v", paymentId, err)
			errorCount++
			continue
		}

		log.Printf("Successfully migrated payment_settings for payment ID %d", paymentId)
		migratedCount++
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Fatal("Failed to commit transaction:", err)
	}

	log.Println("========================================")
	log.Printf("Migration completed!")
	log.Printf("Successfully migrated: %d payments", migratedCount)
	log.Printf("Already migrated/skipped: %d payments", skippedCount)
	log.Printf("Errors encountered: %d payments", errorCount)
	log.Println("========================================")
}

// mapMethodToCoinNetwork maps the old payment method to new coin and network values
func mapMethodToCoinNetwork(method utils.Method) (coin, network string) {
	switch method {
	case utils.PaymentTypeBTC:
		return "BTC", "btc"
	case utils.PaymentTypeLTC:
		return "LTC", "ltc"
	case utils.PaymentTypeDCR:
		return "DCR", "dcr"
	case utils.PaymentTypeETH:
		return "ETH", "erc20" // Default ETH to ERC20 network
	case utils.PaymentTypeUSDT:
		return "USDT", "erc20" // Default USDT to ERC20
	default:
		// Handle unknown types
		methodStr := string(method)
		if methodStr == "usdc" {
			return "USDC", "erc20"
		}
		return "", ""
	}
}

func main() {
	fmt.Println("========================================")
	fmt.Println("Payment Settings Migration Tool")
	fmt.Println("This will migrate payment_settings in payments table")
	fmt.Println("from old format (type, address) to new format (coin, network, address)")
	fmt.Println("========================================")
	
	// Check if DATABASE_URL is set
	if os.Getenv("DATABASE_URL") == "" {
		fmt.Println("\nError: DATABASE_URL environment variable is not set")
		fmt.Println("Usage: DATABASE_URL=postgres://... go run migrate_payments.go")
		os.Exit(1)
	}
	
	fmt.Println("\nPress Enter to continue or Ctrl+C to cancel...")
	fmt.Scanln()
	
	MigratePaymentsSettings()
}