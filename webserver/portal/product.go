package portal

import (
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"gorm.io/gorm"
)

type CreateForm struct {
	UserName       string `validate:"required,alphanum,gte=4,lte=32"`
	DisplayName    string
	Password       string `validate:"required"`
	Email          string `validate:"omitempty,email"`
	DefaultPayment utils.Method
	PaymentAddress string
}

type UpdateProductRequest struct {
	// Sender is the person who will pay for the payment
	Id          uint64  `json:"id"`
	ProductCode string  `json:"productCode"`
	ProductName string  `json:"productName"`
	Description string  `json:"description"`
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
