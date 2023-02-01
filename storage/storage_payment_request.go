package storage

import (
	"code.cryptopower.dev/mgmt-ng/be/payment"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"time"
)

type PaymentStatus int

func (p PaymentStatus) String() string {
	switch p {
	case PaymentStatusCreated:
		return "created"
	case PaymentStatusSent:
		return "sent"
	case PaymentStatusPaid:
		return "paid"
	}
	return "unknown"
}

func (p PaymentStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *PaymentStatus) UnmarshalJSON(v []byte) error {
	var val string
	if err := json.Unmarshal(v, &val); err != nil {
		return err
	}
	switch val {
	case "created":
		*p = PaymentStatusCreated
		return nil
	case "sent":
		*p = PaymentStatusSent
		return nil
	case "paid":
		*p = PaymentStatusPaid
		return nil
	}
	return fmt.Errorf("payment status invalid value")
}

const (
	PaymentStatusCreated PaymentStatus = iota
	PaymentStatusSent
	PaymentStatusPaid
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

func (p *PaymentContact) UnmarshalJSON(v []byte) error {
	var val string
	if err := json.Unmarshal(v, &val); err != nil {
		return err
	}
	switch val {
	case "internal":
		*p = PaymentTypeInternal
		return nil
	case "email":
		*p = PaymentTypeEmail
		return nil
	}
	return fmt.Errorf("payment contact invalid value")
}

const (
	PaymentTypeInternal PaymentContact = iota
	PaymentTypeEmail
)

type Payment struct {
	Id             uint64 `gorm:"primarykey"`
	RequesterId    uint64
	SenderId       uint64
	SenderEmail    string
	Amount         float64
	ConvertRate    float64
	ConvertTime    time.Time
	Description    string
	TxId           string
	Status         PaymentStatus
	PaymentMethod  payment.Method
	PaymentAddress string
	ContactMethod  PaymentContact
	CreatedAt      time.Time
	SentAt         time.Time
	PaidAt         time.Time
}

type PaymentFilter struct {
	Ids []uint64
}

func (f *PaymentFilter) BindQuery(db *gorm.DB) *gorm.DB {
	if len(f.Ids) > 0 {
		db = db.Where("id", f.Ids)
	}
	return db
}
