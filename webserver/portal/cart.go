package portal

import (
	"gorm.io/gorm"
)

type CartForm struct {
	OwnerId   uint64 `json:"ownerId"`
	OwnerName string `json:"ownerName"`
	ProductId uint64 `json:"productId"`
	Quantity  int    `json:"quantity"`
}

type CartDisplayData struct {
	OwnerId      uint64  `json:"ownerId"`
	OwnerName    string  `json:"ownerName"`
	AvatarBase64 string  `json:"avatarBase64"`
	ProductId    uint64  `json:"productId"`
	ProductName  string  `json:"productName"`
	Price        float64 `json:"price"`
	Stock        int     `json:"stock"`
	Currency     string  `json:"currency"`
	Quantity     int     `json:"quantity"`
}

type CartWithList struct {
	List []uint64
}

func (a CartWithList) RequestedSort() string {
	return ""
}
func (a CartWithList) BindQuery(db *gorm.DB) *gorm.DB {
	return db.Where("user_id IN ?", a.List)
}
func (a CartWithList) BindFirst(db *gorm.DB) *gorm.DB {
	return db
}
func (a CartWithList) BindCount(db *gorm.DB) *gorm.DB {
	return db
}
func (a CartWithList) Sortable() map[string]bool {
	return map[string]bool{}
}
