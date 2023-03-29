package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/payment"
	"gorm.io/gorm"
)

const (
	PaymentTypeRequest  = "request"
	PaymentTypeReminder = "reminder"
)

type PaymentStatus int

func (p PaymentStatus) String() string {
	switch p {
	case PaymentStatusCreated:
		return "draft"
	case PaymentStatusSent:
		return "sent"
	case PaymentStatusConfirmed:
		return "confirmed"
	case PaymentStatusPaid:
		return "paid"
	case PaymentStatusWaitApproval:
		return "wait approve"
	case PaymentStatusApproved:
		return "approved"
	case PaymentStatusRejected:
		return "rejected"
	}
	return "unknown"
}

func (p PaymentStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *PaymentStatus) UnmarshalText(val []byte) error {
	switch string(val) {
	case "draft":
		*p = PaymentStatusCreated
	case "sent":
		*p = PaymentStatusSent
	case "confirmed":
		*p = PaymentStatusConfirmed
	case "paid":
		*p = PaymentStatusPaid
	case "wait approve":
		*p = PaymentStatusWaitApproval
	case "approved":
		*p = PaymentStatusApproved
	}
	return nil
}

func (p *PaymentStatus) UnmarshalJSON(v []byte) error {
	var val string
	if err := json.Unmarshal(v, &val); err != nil {
		return err
	}
	return p.UnmarshalText([]byte(val))
}

const (
	PaymentStatusCreated PaymentStatus = iota
	PaymentStatusSent
	PaymentStatusConfirmed
	PaymentStatusPaid
	PaymentStatusWaitApproval
	PaymentStatusApproved
	PaymentStatusRejected
)

type PaymentContact int

func (p PaymentContact) String() string {
	switch p {
	case PaymentTypeInternal:
		return "internal"
	case PaymentTypeEmail:
		return "email"
	}
	return "unknown"
}

func (p PaymentContact) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}
func (p *PaymentContact) UnmarshalText(val []byte) error {
	switch string(val) {
	case "internal":
		*p = PaymentTypeInternal
		return nil
	case "email":
		*p = PaymentTypeEmail
		return nil
	}
	return fmt.Errorf("payment contact invalid value")
}
func (p *PaymentContact) UnmarshalJSON(v []byte) error {
	var val string
	if err := json.Unmarshal(v, &val); err != nil {
		return err
	}
	return p.UnmarshalText([]byte(val))
}

const (
	PaymentTypeInternal PaymentContact = iota
	PaymentTypeEmail
)

type PaymentDetail struct {
	Hours       float64 `json:"hours"`
	Cost        float64 `json:"cost"`
	Description string  `json:"description"`
}

type PaymentDetails []PaymentDetail

// Value Marshal
func (a PaymentDetails) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan Unmarshal
func (a *PaymentDetails) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

type Payment struct {
	Id              uint64          `gorm:"primarykey" json:"id"`
	CreatorId       uint64          `json:"creatorId"`
	SenderId        uint64          `json:"senderId"`
	SenderName      string          `json:"senderName" gorm:"->"`
	ReceiverId      uint64          `json:"receiverId"`
	ReceiverName    string          `json:"receiverName" gorm:"->"`
	ExternalEmail   string          `json:"externalEmail"`
	Amount          float64         `json:"amount"`
	HourlyRate      float64         `json:"hourlyRate"`
	PaymentSettings PaymentSettings `json:"paymentSettings" gorm:"type:jsonb"`
	Approvers       Approvers       `json:"approvers" gorm:"type:jsonb"`
	Details         PaymentDetails  `json:"details" gorm:"type:jsonb"`
	ConvertRate     float64         `json:"convertRate"`
	ConvertTime     time.Time       `json:"convertTime"`
	ExpectedAmount  float64         `json:"expectedAmount"`
	TxId            string          `json:"txId"`
	Status          PaymentStatus   `json:"status"`
	PaymentMethod   payment.Method  `json:"paymentMethod"`
	PaymentAddress  string          `json:"paymentAddress"`
	ContactMethod   PaymentContact  `json:"contactMethod"`
	RejectionReason string          `json:"rejectionReason"`
	CreatedAt       time.Time       `json:"createdAt"`
	SentAt          time.Time       `json:"sentAt"`
	PaidAt          time.Time       `json:"paidAt"`
	IsApproved      bool            `json:"isApproved" gorm:"->"`
}

type PaymentFilter struct {
	Sort
	RequestType    string           `schema:"requestType"`
	Ids            []uint64         `schema:"ids"`
	ReceiverIds    []uint64         `schema:"receiverIds"`
	SenderIds      []uint64         `schema:"senderIds"`
	Statuses       []PaymentStatus  `schema:"statuses"`
	ContactMethods []PaymentContact `schema:"contactMethods"`
	Approvers      []ApproverSettings
}

func (f *PaymentFilter) selectFields(db *gorm.DB) *gorm.DB {
	return db.Select("payments.*, receiver.user_name as receiver_name, sender.user_name as sender_name").
		Joins("left join users receiver on payments.receiver_id = receiver.id").
		Joins("left join users sender on payments.sender_id = sender.id")
}

func (f *PaymentFilter) BindCount(db *gorm.DB) *gorm.DB {
	if len(f.Ids) > 0 {
		if f.RequestType == PaymentTypeReminder {
			db = db.Or("payments.id", f.Ids)
		} else {
			db = db.Where("payments.id", f.Ids)
		}
	}
	if len(f.ReceiverIds) > 0 && len(f.SenderIds) > 0 {
		db = db.Where("receiver_id IN ? OR sender_id IN ?", f.ReceiverIds, f.SenderIds)
	} else {
		if len(f.ReceiverIds) > 0 {
			db = db.Where("receiver_id IN ?", f.ReceiverIds)
		}
		if len(f.SenderIds) > 0 {
			db = db.Where("sender_id IN ?", f.SenderIds)
		}
	}

	if len(f.Statuses) > 0 {
		db = db.Where("payments.status IN ?", f.Statuses)
	}
	if len(f.ContactMethods) > 0 {
		db = db.Where("contact_method IN ?", f.ContactMethods)
	}

	if f.RequestType == PaymentTypeReminder && len(f.Approvers) > 0 {
		for _, setting := range f.Approvers {
			db = db.Or("receiver_id = ? AND sender_id = ?", setting.RecipientId, setting.SendUserId)
		}
	}

	return db
}

func (f *PaymentFilter) BindQuery(db *gorm.DB) *gorm.DB {
	db = f.selectFields(db)
	db = f.Sort.BindQuery(db)
	return f.BindCount(db)
}

func (f *PaymentFilter) BindFirst(db *gorm.DB) *gorm.DB {
	db = f.selectFields(db)
	if len(f.Ids) > 0 {
		db = db.Where("payments.id", f.Ids)
	}
	return db
}

func (f *PaymentFilter) Sortable() map[string]bool {
	return map[string]bool{
		"createdAt": true,
		"paidAt":    true,
		"status":    true,
		"amount":    true,
	}
}
