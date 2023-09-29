package portal

import (
	"gorm.io/gorm"
)

type CreateProductForm struct {
	ProductCode string
	ProductName string `validate:"required"`
	Description string
	Currency    string `validate:"required"`
	Avatar      string
	Images      string
	Price       float32 `validate:"required"`
	Stock       int     `validate:"required"`
}

type UpdateProductRequest struct {
	// Sender is the person who will pay for the payment
	Id          uint64  `json:"id"`
	ProductCode string  `json:"productCode"`
	ProductName string  `json:"productName"`
	Description string  `json:"description"`
	OwnerId     uint    `json:"ownerId"`
	Currency    string  `json:"currency"`
	Avatar      string  `json:"avatar"`
	Images      string  `json:"images"`
	Price       float32 `json:"price"`
	Stock       int     `json:"stock"`
	Status      uint    `json:"status"`
}

type ProductWithList struct {
	List []uint64
}

func (a ProductWithList) RequestedSort() string {
	return ""
}
func (a ProductWithList) BindQuery(db *gorm.DB) *gorm.DB {
	return db.Where("id IN ?", a.List)
}
func (a ProductWithList) BindFirst(db *gorm.DB) *gorm.DB {
	return db
}
func (a ProductWithList) BindCount(db *gorm.DB) *gorm.DB {
	return db
}
func (a ProductWithList) Sortable() map[string]bool {
	return map[string]bool{}
}
