package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type OrderStorage interface {
	CreateOrder(order *Order) error
	UpdateOrder(order *Order) error
	QueryOrder(field string, val interface{}) (*Order, error)
	QueryOrderWithList(field string, val interface{}) ([]Order, error)
}

type ProductPayments []ProductPayment
type ProductPaymentsDisplay []ProductPaymentDisplay

// Value Marshal
func (a ProductPayments) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan Unmarshal
func (a *ProductPayments) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

type Order struct {
	OrderId         uint64          `json:"orderId" gorm:"primaryKey"`
	OrderCode       string          `json:"orderCode"`
	UserId          uint64          `json:"userId"`
	UserName        string          `json:"userName"`
	OwnerId         uint64          `json:"ownerId"`
	OwnerName       string          `json:"ownerName"`
	ShopName        string          `json:"shopName"`
	ProductPayments ProductPayments `json:"productPayments" gorm:"type:jsonb"`
	PhoneNumber     string          `json:"phoneNumber"`
	Address         string          `json:"address"`
	Memo            string          `json:"memo"`
	PaymentId       uint64          `json:"paymentId"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
}

type ProductPayment struct {
	ProductId   uint64  `json:"productId"`
	ProductName string  `json:"productName"`
	Avatar      string  `json:"avatar"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	Currency    string  `json:"currency"`
	Amount      float64 `json:"amount"`
}

type ProductPaymentDisplay struct {
	ProductId    uint64  `json:"productId"`
	ProductName  string  `json:"productName"`
	AvatarBase64 string  `json:"avatarBase64"`
	Price        float64 `json:"price"`
	Quantity     int     `json:"quantity"`
	Currency     string  `json:"currency"`
	Amount       float64 `json:"amount"`
}

func (Order) TableName() string {
	return "order"
}

func (p *psql) CreateOrder(order *Order) error {
	return p.db.Create(order).Error
}

func (p *psql) UpdateOrder(order *Order) error {
	return p.db.Save(order).Error
}

func (p *psql) QueryOrder(field string, val interface{}) (*Order, error) {
	var order Order
	var err = p.db.Where(fmt.Sprintf("%s = ?", field), val).First(&order).Error
	return &order, err
}

func (p *psql) QueryOrderWithList(field string, val interface{}) ([]Order, error) {
	var orders []Order
	var err = p.db.Where(fmt.Sprintf("%s IN ?", field), val).Find(&orders).Error
	return orders, err
}

type OrderFilter struct {
	Sort
	UserId uint64
}

func (f *OrderFilter) BindQuery(db *gorm.DB) *gorm.DB {
	db = f.Sort.BindQuery(db)
	return f.BindCount(db)
}

func (f *OrderFilter) BindCount(db *gorm.DB) *gorm.DB {
	if f.UserId > 0 {
		db = db.Where("user_id", f.UserId)
	}
	return db
}

func (f *OrderFilter) BindFirst(db *gorm.DB) *gorm.DB {
	if f.UserId > 0 {
		db = db.Where("user_id", f.UserId)
	}
	return db
}

func (f *OrderFilter) Sortable() map[string]bool {
	return map[string]bool{
		"updatedAt": true,
	}
}
