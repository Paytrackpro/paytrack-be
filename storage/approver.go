package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"code.cryptopower.dev/mgmt-ng/be/payment"
)

type PaymentSetting struct {
	Type      payment.Method `json:"type"`
	Address   string         `json:"address"`
	IsDefault bool           `json:"isDefault"`
}

type PaymentSettings []PaymentSetting

type Approvers []Approver

type Approver struct {
	ApproverId   uint64 `json:"approverId"`
	ApproverName string `json:"approverName"`
	Status       uint64 `json:"status"`
}

// Value Marshal
func (a Approvers) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan Unmarshal
func (a *Approvers) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

type ApproverSettings struct {
	Id           uint64 `gorm:"primarykey" json:"id"`
	ApproverId   uint64 `json:"approverId"`
	SendUserId   uint64 `json:"sendUserId"`
	RecipientId  uint64 `json:"recipientId"`
	ApproverName string `json:"approverName"`
	SendUserName string `json:"sendUserName"`
}
