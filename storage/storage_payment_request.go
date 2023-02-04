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
	case PaymentStatusPaid:
		return "paid"
	}
	return "unknown"
}

func (p PaymentStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *PaymentStatus) UnmarshalText(val []byte) error {
	switch string(val) {
	case "created":
		*p = PaymentStatusCreated
		return nil
	case "paid":
		*p = PaymentStatusPaid
		return nil
	}
	return fmt.Errorf("payment status invalid value")
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
	PaidAt         time.Time
}

type PaymentFilter struct {
	Sort
	Ids            []uint64
	RequesterIds   []uint64
	SenderIds      []uint64
	Statuses       []PaymentStatus
	ContactMethods []PaymentContact
}

func (f *PaymentFilter) BindQuery(db *gorm.DB) *gorm.DB {
	db = f.Sort.BindQuery(db)
	if len(f.Ids) > 0 {
		db = db.Where("id", f.Ids)
	}
	if len(f.RequesterIds) > 0 && len(f.SenderIds) > 0 {
		db = db.Where("requester_id IN ? OR sender_id IN ?", f.RequesterIds, f.SenderIds)
	} else {
		if len(f.RequesterIds) > 0 {
			db = db.Where("requester_id", f.SenderIds)
		}
		if len(f.SenderIds) > 0 {
			db = db.Where("sender_id", f.SenderIds)
		}
	}

	if len(f.Statuses) > 0 {
		db = db.Where("status", f.Statuses)
	}
	if len(f.ContactMethods) > 0 {
		db = db.Where("contact_method", f.ContactMethods)
	}
	return db
}

func (f *PaymentFilter) Sortable() []string {
	return []string{
		"CreatedAt",
		"PaidAt",
		"Status",
	}
}
