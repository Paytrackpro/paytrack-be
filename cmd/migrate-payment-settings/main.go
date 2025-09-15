package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

	// Start migration
	log.Println("Starting payment settings migration...")
	
	// Begin transaction
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Fatal("Migration failed with panic:", r)
		}
	}()

	// Get all users with payment settings
	var users []storage.User
	if err := tx.Find(&users).Error; err != nil {
		tx.Rollback()
		log.Fatal("Failed to fetch users:", err)
	}

	migratedCount := 0
	for _, user := range users {
		// Skip users without payment settings
		if len(user.PaymentSettings) == 0 {
			continue
		}

		// Process each payment setting
		for _, setting := range user.PaymentSettings {
			// Map payment type to coin and network code
			coin, networkCode := mapPaymentTypeToCoinNetwork(setting.Type)
			if coin == "" || networkCode == "" {
				log.Printf("Skipping unknown payment type %s for user %d", setting.Type.String(), user.Id)
				continue
			}

			// Create payment method
			paymentMethod := storage.UserPaymentMethod{
				UserId:    user.Id,
				Label:     fmt.Sprintf("Legacy %s wallet", coin),
				Coin:      coin,
				Network:   networkCode, // Store network code instead of display name
				Address:   setting.Address,
				CreatedAt: user.CreatedAt, // Use user's creation time
				UpdatedAt: time.Now(),
			}

			if err := tx.Create(&paymentMethod).Error; err != nil {
				tx.Rollback()
				log.Fatalf("Failed to create payment method for user %d: %v", user.Id, err)
			}

			log.Printf("Migrated %s payment method for user %d (network: %s)", coin, user.Id, networkCode)
			migratedCount++
		}
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		log.Fatal("Failed to commit transaction:", err)
	}

	log.Printf("Migration completed successfully. Migrated %d payment methods.", migratedCount)
}

// mapPaymentTypeToCoinNetwork maps the old payment type to new coin and network code values
func mapPaymentTypeToCoinNetwork(paymentType utils.Method) (coin, networkCode string) {
	switch paymentType {
	case utils.PaymentTypeBTC:
		return "BTC", "btc" // Use network code instead of display name
	case utils.PaymentTypeLTC:
		return "LTC", "ltc"
	case utils.PaymentTypeDCR:
		return "DCR", "dcr"
	case utils.PaymentTypeETH:
		return "ETH", "erc20" // Default ETH to ERC20 network
	case utils.PaymentTypeUSDT:
		return "USDT", "erc20" // Default USDT to ERC20 network
	default:
		return "", ""
	}
}