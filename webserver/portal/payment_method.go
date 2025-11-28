package portal

import (
	"github.com/Paytrackpro/paytrack-be/storage"
	"github.com/Paytrackpro/paytrack-be/utils"
	"gorm.io/gorm"
)

// CreatePaymentMethodRequest represents the request to create a new payment method
type CreatePaymentMethodRequest struct {
	Label   string `json:"label" validate:"required"`
	Coin    string `json:"coin" validate:"required"`
	Network string `json:"network" validate:"required"` // Network code (e.g., "btc", "erc20")
	Address string `json:"address" validate:"required"`
}

// UpdatePaymentMethodRequest represents the request to update a payment method (only label can be updated)
type UpdatePaymentMethodRequest struct {
	Label string `json:"label" validate:"required"`
}

// ValidateAddressRequest represents the request to validate a wallet address
type ValidateAddressRequest struct {
	Coin    string `json:"coin" validate:"required"`
	Network string `json:"network" validate:"required"` // Network code (e.g., "btc", "erc20")
	Address string `json:"address" validate:"required"`
}

// ValidateAddressResponse represents the response from address validation
type ValidateAddressResponse struct {
	IsValid bool   `json:"isValid"`
	Reason  string `json:"reason,omitempty"`
}

// SupportedNetworksResponse represents the response for supported networks API
type SupportedNetworksResponse struct {
	SupportedCoins []utils.CoinNetworkSupport `json:"supportedCoins"`
}

// SettingsResponse represents the comprehensive settings response for clients
type SettingsResponse struct {
	SupportedCoins []utils.CoinNetworkSupport `json:"supportedCoins"`
	// Add other settings here as needed in the future
}

// PaymentMethodFilter represents filter options for listing payment methods
type PaymentMethodFilter struct {
	storage.Sort
	UserId uint64 `json:"userId"`
}

func (f *PaymentMethodFilter) BindQuery(db *gorm.DB) *gorm.DB {
	db = f.Sort.BindQuery(db)
	return f.BindCount(db)
}

func (f *PaymentMethodFilter) BindCount(db *gorm.DB) *gorm.DB {
	if f.UserId > 0 {
		db = db.Where("user_id = ?", f.UserId)
	}
	return db
}

func (f *PaymentMethodFilter) BindFirst(db *gorm.DB) *gorm.DB {
	return db
}

func (f *PaymentMethodFilter) Sortable() map[string]bool {
	return map[string]bool{
		"createdAt": true,
		"updatedAt": true,
		"label":     true,
		"coin":      true,
		"network":   true,
	}
}
