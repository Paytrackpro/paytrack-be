package storage

import (
	"fmt"
	"strings"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/utils"
	"gorm.io/gorm"
)

type ProductStorage interface {
	CreateProduct(product *Product) error
	UpdateProduct(product *Product) error
	QueryProduct(field string, val interface{}) (*Product, error)
	QueryProductWithList(field string, val interface{}) ([]Product, error)
}

type Product struct {
	// Sender is the person who will pay for the payment
	Id          uint64    `json:"id" gorm:"primarykey"`
	ProductCode string    `json:"productCode"`
	ProductName string    `json:"productName"`
	Description string    `json:"description"`
	Currency    string    `json:"currency"`
	Avatar      string    `json:"avatar"`
	Images      string    `json:"images"`
	OwnerId     uint64    `json:"ownerId"`
	OwnerName   string    `json:"ownerName"`
	ShopName    string    `json:"shopName"`
	Price       float64   `json:"price"`
	Stock       int       `json:"stock"`
	Status      uint32    `json:"status"`
	CreatedAt   time.Time `json:"createdAt" gorm:"default:current_timestamp"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
type StoreInfo struct {
	OwnerId   uint64 `json:"ownerId"`
	OwnerName string `json:"ownerName"`
	ShopName  string `json:"shopName"`
	Count     int    `json:"count"`
}
type ProductStatus int

const (
	ProductHidden PaymentStatus = iota
	ProductActive
)

func (Product) TableName() string {
	return "products"
}

func (p *psql) CreateProduct(product *Product) error {
	return p.db.Create(product).Error
}

func (p *psql) UpdateProduct(product *Product) error {
	return p.db.Save(product).Error
}

func (p *psql) QueryProduct(field string, val interface{}) (*Product, error) {
	var product Product
	var err = p.db.Where(fmt.Sprintf("%s = ?", field), val).First(&product).Error
	return &product, err
}

func (p *psql) QueryProductWithList(field string, val interface{}) ([]Product, error) {
	var product []Product
	var err = p.db.Where(fmt.Sprintf("%s IN ?", field), val).Find(&product).Error
	return product, err
}

type ProductFilter struct {
	Sort
	KeySearch   string
	ProductCode string
	OwnerId     uint64
}

func (f *ProductFilter) BindQuery(db *gorm.DB) *gorm.DB {
	db = f.Sort.BindQuery(db)
	return f.BindCount(db)
}

func (f *ProductFilter) BindCount(db *gorm.DB) *gorm.DB {
	if !utils.IsEmpty(f.KeySearch) {
		keySearch := fmt.Sprintf("%%%s%%", strings.TrimSpace(f.KeySearch))
		db = db.Where("product_name LIKE ?", keySearch)
	}
	if f.OwnerId > 0 {
		db = db.Where("owner_id", f.OwnerId)
	}
	return db
}

func (f *ProductFilter) BindFirst(db *gorm.DB) *gorm.DB {
	if len(f.ProductCode) > 0 {
		db = db.Where("product_code LIKE ? ", f.ProductCode)
	}
	if f.OwnerId > 0 {
		db = db.Where("owner_id", f.OwnerId)
	}
	return db
}

func (f *ProductFilter) Sortable() map[string]bool {
	return map[string]bool{
		"productName": true,
		"price":       true,
		"createdAt":   true,
	}
}
