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
	Price       float64 `validate:"required"`
	Stock       int     `validate:"required"`
}

// Sender is the person who will pay for the payment
type UpdateProductRequest struct {
	Id          uint64  `json:"id"`
	ProductCode string  `json:"productCode"`
	ProductName string  `json:"productName"`
	Description string  `json:"description"`
	OwnerId     uint32  `json:"ownerId"`
	Currency    string  `json:"currency"`
	Avatar      string  `json:"avatar"`
	Images      string  `json:"images"`
	Price       float64 `json:"price"`
	Stock       int     `json:"stock"`
	Status      uint32  `json:"status"`
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
