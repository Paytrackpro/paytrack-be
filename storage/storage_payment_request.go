package storage

import (
	"code.cryptopower.dev/mgmt-ng/be/payment"
	"encoding/json"
	"fmt"
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
	switch string(v) {
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
	return fmt.Errorf("invalid value")
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
	switch string(v) {
	case "internal":
		*p = PaymentTypeInternal
		return nil
	case "email":
		*p = PaymentTypeEmail
		return nil
	}
	return fmt.Errorf("invalid value")
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
