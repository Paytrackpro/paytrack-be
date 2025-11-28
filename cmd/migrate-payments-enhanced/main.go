package main

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/Paytrackpro/paytrack-be/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OldPaymentSetting represents the current structure without network field
type OldPaymentSetting struct {
	Type    utils.Method `json:"type"`
	Address string       `json:"address"`
}

// EnhancedPaymentSetting represents the new structure with network field
// BUT also includes the old "type" field for backward compatibility
type EnhancedPaymentSetting struct {
	Type    utils.Method `json:"type"`    // Keep for backward compatibility with queries
	Coin    string       `json:"coin"`    // New field for coin type
	Network string       `json:"network"` // New field for network
	Address string       `json:"address"`
}

type EnhancedPaymentSettings []EnhancedPaymentSetting

// Value Marshal for EnhancedPaymentSettings
func (ps EnhancedPaymentSettings) Value() (driver.Value, error) {
	return json.Marshal(ps)
}

// Scan Unmarshal for EnhancedPaymentSettings
func (ps *EnhancedPaymentSettings) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	b, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed")
	}
	return json.Unmarshal(b, ps)
}

// MigratePaymentsSettingsEnhanced migrates payment_settings with backward compatibility
func MigratePaymentsSettingsEnhanced() {
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

	// Begin transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Fatal("Migration failed with panic:", r)
		}
	}()

	// First, analyze current state
	analyzeCurrentState(tx)

	// Get all payment IDs and settings into memory
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
	rows.Close()

	log.Printf("Found %d payments with payment_settings to process", len(payments))

	migratedCount := 0
	skippedCount := 0
	errorCount := 0

	// Process each payment
	for _, payment := range payments {
		paymentId := payment.ID
		paymentSettingsJSON := payment.Settings

		// Check if already migrated (has "coin" field)
		if strings.Contains(string(paymentSettingsJSON), `"coin"`) {
			log.Printf("Payment %d already migrated, skipping", paymentId)
			skippedCount++
			continue
		}

		// Parse old payment settings
		var oldSettings []OldPaymentSetting
		if err := json.Unmarshal(paymentSettingsJSON, &oldSettings); err != nil {
			log.Printf("Error parsing payment_settings for payment %d: %v", paymentId, err)
			errorCount++
			continue
		}

		// Convert to new format with backward compatibility
		newSettings := make([]EnhancedPaymentSetting, 0, len(oldSettings))

		for _, oldSetting := range oldSettings {
			coin, network := mapMethodToCoinNetwork(oldSetting.Type)

			if coin == "" || network == "" {
				log.Printf("Warning: Unknown payment type %s for payment %d, keeping original",
					oldSetting.Type.String(), paymentId)
				// Keep original type for unknown methods
				coin = strings.ToUpper(oldSetting.Type.String())
				network = "unknown"
			}

			newSettings = append(newSettings, EnhancedPaymentSetting{
				Type:    oldSetting.Type, // IMPORTANT: Keep original type for backward compatibility
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

	// Verify migration results
	verifyMigration(tx)

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Fatal("Failed to commit transaction:", err)
	}
}

// analyzeCurrentState analyzes the current state of payment_settings
func analyzeCurrentState(db *gorm.DB) {

	// Count payments with different payment types
	type TypeCount struct {
		Type  string
		Count int
	}

	var counts []TypeCount
	query := `
		SELECT 
			jsonb_array_elements(payment_settings)->>'type' as type,
			COUNT(*) as count
		FROM payments
		WHERE payment_settings IS NOT NULL 
		AND payment_settings::text != 'null' 
		AND payment_settings::text != '[]'
		GROUP BY type
	`

	if err := db.Raw(query).Scan(&counts).Error; err != nil {
		log.Printf("Could not analyze payment types: %v", err)
	} else {
		log.Println("Payment types distribution:")
		for _, tc := range counts {
			log.Printf("  %s: %d payments", tc.Type, tc.Count)
		}
	}

	// Check for already migrated payments (have "coin" field)
	var migratedCount int64
	db.Raw(`
		SELECT COUNT(*) 
		FROM payments 
		WHERE payment_settings::text LIKE '%"coin"%'
	`).Scan(&migratedCount)

	log.Printf("Already migrated payments: %d", migratedCount)
}

// verifyMigration verifies the migration results
func verifyMigration(db *gorm.DB) {
	log.Println("\n--- Verifying Migration Results ---")

	// Test backward compatibility query
	var btcCount int64
	testQuery := `SELECT COUNT(*) FROM payments WHERE payment_settings @> '[{"type": "btc"}]'`
	if err := db.Raw(testQuery).Scan(&btcCount).Error; err != nil {
		log.Printf("WARNING: Backward compatibility test failed: %v", err)
	} else {
		log.Printf("Backward compatibility OK: Found %d BTC payments using old query", btcCount)
	}

	// Test new query format
	var btcCountNew int64
	newQuery := `SELECT COUNT(*) FROM payments WHERE payment_settings @> '[{"coin": "BTC"}]'`
	if err := db.Raw(newQuery).Scan(&btcCountNew).Error; err != nil {
		log.Printf("New query test failed: %v", err)
	} else {
		log.Printf("New format OK: Found %d BTC payments using new query", btcCountNew)
	}
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
	MigratePaymentsSettingsEnhanced()
}
