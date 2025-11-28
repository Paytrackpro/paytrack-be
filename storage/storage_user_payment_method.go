package storage

import (
	"time"

	"github.com/Paytrackpro/paytrack-be/utils"
)

// UserPaymentMethod represents a user's payment method
type UserPaymentMethod struct {
	Id        uint64    `json:"id" gorm:"primarykey"`
	UserId    uint64    `json:"userId" gorm:"index"`
	Label     string    `json:"label"`
	Coin      string    `json:"coin"`
	Network   string    `json:"network"` // Stores network code (e.g., "btc", "erc20")
	Address   string    `json:"address"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// GetNetworkInfo returns the network information with both code and display name
func (upm *UserPaymentMethod) GetNetworkInfo() utils.NetworkInfo {
	network := utils.NetworkFromCode(upm.Network)
	return network.Info()
}

func (UserPaymentMethod) TableName() string {
	return "user_payment_methods"
}
