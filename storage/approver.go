package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"code.cryptopower.dev/mgmt-ng/be/utils"
)

type PaymentSetting struct {
	Type    utils.Method `json:"type"`
	Address string       `json:"address"`
}

type PaymentSettings []PaymentSetting

type Approvers []Approver

type Approver struct {
	ApproverId   uint64 `json:"approverId"`
	ApproverName string `json:"approverName"`
	IsApproved   bool   `json:"isApproved"`
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
