package storage

import (
	"code.cryptopower.dev/mgmt-ng/be/payment"
	"time"
)

type PaymentStatus int

const (
	PaymentStatusCreated PaymentStatus = iota
	PaymentStatusSent
	PaymentStatusPaid
)

type PaymentContact int

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
