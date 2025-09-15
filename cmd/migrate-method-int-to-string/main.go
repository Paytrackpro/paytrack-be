package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// LegacyPaymentSetting represents the old payment setting structure with int Method
type LegacyPaymentSetting struct {
	Type    int    `json:"type"`
	Address string `json:"address"`
}

// NewPaymentSetting represents the new payment setting structure with string Method
type NewPaymentSetting struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

func main() {
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

	log.Println("Starting Method type migration from int to string...")

	// Start transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Fatal("Migration failed with panic:", r)
		}
	}()

	// Update users.payment_settings JSONB field
	if err := migrateUsersPaymentSettings(tx); err != nil {
		tx.Rollback()
		log.Fatal("Failed to migrate users payment_settings:", err)
	}

	// Update payments.payment_settings JSONB field
	if err := migratePaymentsPaymentSettings(tx); err != nil {
		tx.Rollback()
		log.Fatal("Failed to migrate payments payment_settings:", err)
	}

	// Update payments.payment_method field
	if err := migratePaymentsPaymentMethod(tx); err != nil {
		tx.Rollback()
		log.Fatal("Failed to migrate payments payment_method:", err)
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Fatal("Failed to commit transaction:", err)
	}

	log.Println("Method type migration completed successfully!")
}

func migrateUsersPaymentSettings(db *gorm.DB) error {
	log.Println("Migrating users.payment_settings...")

	// Get all users with payment_settings
	rows, err := db.Raw("SELECT id, payment_settings FROM users WHERE payment_settings IS NOT NULL AND payment_settings::text != 'null'").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var userId uint64
		var paymentSettingsJSON []byte

		if err := rows.Scan(&userId, &paymentSettingsJSON); err != nil {
			return err
		}

		// Parse old format
		var legacySettings []LegacyPaymentSetting
		if err := json.Unmarshal(paymentSettingsJSON, &legacySettings); err != nil {
			log.Printf("Skipping user %d: could not parse payment_settings: %v", userId, err)
			continue
		}

		// Convert to new format
		var newSettings []NewPaymentSetting
		for _, legacy := range legacySettings {
			newType := convertIntMethodToString(legacy.Type)
			if newType != "" {
				newSettings = append(newSettings, NewPaymentSetting{
					Type:    newType,
					Address: legacy.Address,
				})
			}
		}

		// Update database
		newJSON, err := json.Marshal(newSettings)
		if err != nil {
			return err
		}

		if err := db.Exec("UPDATE users SET payment_settings = ? WHERE id = ?", string(newJSON), userId).Error; err != nil {
			return err
		}

		log.Printf("Updated user %d payment_settings", userId)
		count++
	}

	log.Printf("Migrated %d users payment_settings", count)
	return nil
}

func migratePaymentsPaymentSettings(db *gorm.DB) error {
	log.Println("Migrating payments.payment_settings...")

	// Get all payments with payment_settings
	rows, err := db.Raw("SELECT id, payment_settings FROM payments WHERE payment_settings IS NOT NULL AND payment_settings::text != 'null'").Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var paymentId uint64
		var paymentSettingsJSON []byte

		if err := rows.Scan(&paymentId, &paymentSettingsJSON); err != nil {
			return err
		}

		// Parse old format
		var legacySettings []LegacyPaymentSetting
		if err := json.Unmarshal(paymentSettingsJSON, &legacySettings); err != nil {
			log.Printf("Skipping payment %d: could not parse payment_settings: %v", paymentId, err)
			continue
		}

		// Convert to new format
		var newSettings []NewPaymentSetting
		for _, legacy := range legacySettings {
			newType := convertIntMethodToString(legacy.Type)
			if newType != "" {
				newSettings = append(newSettings, NewPaymentSetting{
					Type:    newType,
					Address: legacy.Address,
				})
			}
		}

		// Update database
		newJSON, err := json.Marshal(newSettings)
		if err != nil {
			return err
		}

		if err := db.Exec("UPDATE payments SET payment_settings = ? WHERE id = ?", string(newJSON), paymentId).Error; err != nil {
			return err
		}

		log.Printf("Updated payment %d payment_settings", paymentId)
		count++
	}

	log.Printf("Migrated %d payments payment_settings", count)
	return nil
}

func migratePaymentsPaymentMethod(db *gorm.DB) error {
	log.Println("Migrating payments.payment_method...")

	// Map int values to string values for payment_method field
	updates := []struct {
		oldValue int
		newValue string
	}{
		{0, ""},    // PaymentTypeNotSet
		{1, "btc"}, // PaymentTypeBTC
		{2, "ltc"}, // PaymentTypeLTC
		{3, "dcr"}, // PaymentTypeDCR
		// Note: ETH and USDT would be new, so no migration needed for them
	}

	totalUpdated := 0
	for _, update := range updates {
		result := db.Exec("UPDATE payments SET payment_method = ? WHERE payment_method = ?", update.newValue, fmt.Sprintf("%d", update.oldValue))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			log.Printf("Updated %d payments: payment_method %d -> '%s'", result.RowsAffected, update.oldValue, update.newValue)
			totalUpdated += int(result.RowsAffected)
		}
	}

	log.Printf("Migrated %d total payment_method values", totalUpdated)
	return nil
}

// convertIntMethodToString converts old int Method values to new string values
func convertIntMethodToString(oldValue int) string {
	switch oldValue {
	case 0:
		return "" // PaymentTypeNotSet
	case 1:
		return "btc" // PaymentTypeBTC
	case 2:
		return "ltc" // PaymentTypeLTC
	case 3:
		return "dcr" // PaymentTypeDCR
	default:
		return ""
	}
}
