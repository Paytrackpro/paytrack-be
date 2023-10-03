package storage

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type CartStorage interface {
	CreateProduct(product *Product) error
	UpdateProduct(product *Product) error
	QueryProduct(field string, val interface{}) (*Product, error)
	QueryProductWithList(field string, val interface{}) ([]Product, error)
}

type Cart struct {
	// Sender is the person who will pay for the payment
	UserId    uint64    `json:"id"`
	OwnerId   uint64    `json:"ownerId"`
	OwnerName string    `json:"ownerName"`
	UpdatedAt time.Time `json:"updatedAt"`
	ProductId uint64    `json:"productId"`
	Quantity  int       `json:"quantity"`
}

func (Cart) TableName() string {
	return "cart"
}

func (p *psql) AddToCart(cart *Cart) error {
	return p.db.Create(cart).Error
}

func (p *psql) UpdateCart(cart *Cart) error {
	return p.db.Save(cart).Error
}

func (p *psql) QueryCart(field string, val interface{}) (*Cart, error) {
	var cart Cart
	var err = p.db.Where(fmt.Sprintf("%s = ?", field), val).First(&cart).Error
	return &cart, err
}

func (p *psql) QueryCartWithList(field string, val interface{}) ([]Cart, error) {
	var carts []Cart
	var err = p.db.Where(fmt.Sprintf("%s IN ?", field), val).Find(&carts).Error
	return carts, err
}

type CartFilter struct {
	Sort
	UserId uint64
}

func (f *CartFilter) BindQuery(db *gorm.DB) *gorm.DB {
	db = f.Sort.BindQuery(db)
	return f.BindCount(db)
}

func (f *CartFilter) BindCount(db *gorm.DB) *gorm.DB {
	if f.UserId > 0 {
		db = db.Where("user_id", f.UserId)
	}
	return db
}

func (f *CartFilter) BindFirst(db *gorm.DB) *gorm.DB {
	if f.UserId > 0 {
		db = db.Where("user_id", f.UserId)
	}
	return db
}

func (f *CartFilter) Sortable() map[string]bool {
	return map[string]bool{
		"updatedAt": true,
	}
}
