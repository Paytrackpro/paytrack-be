package portal

import (
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"gorm.io/gorm"
)

type OrderForm struct {
	OrderData []OrderData `json:"orderData"`
}

type OrderData struct {
	OwnerId         uint64                  `json:"ownerId"`
	OrderCode       string                  `json:"orderCode"`
	OwnerName       string                  `json:"ownerName"`
	PhoneNumber     string                  `json:"phoneNumber"`
	Address         string                  `json:"address"`
	Memo            string                  `json:"memo"`
	ProductPayments storage.ProductPayments `json:"productPayments"`
}

type OrderDisplayData struct {
	OrderId                uint64                         `json:"orderId"`
	OrderCode              string                         `json:"orderCode"`
	PaymentStatus          int                            `json:"paymentStatus"`
	UserName               string                         `json:"userName"`
	OwnerName              string                         `json:"ownerName"`
	PhoneNumber            string                         `json:"phoneNumber"`
	Address                string                         `json:"address"`
	Memo                   string                         `json:"memo"`
	CreatedAt              time.Time                      `json:"createdAt"`
	ProductPaymentsDisplay storage.ProductPaymentsDisplay `json:"productPaymentsDisplay"`
}

type OrderWithList struct {
	List []uint64
}

func (a OrderWithList) RequestedSort() string {
	return ""
}
func (a OrderWithList) BindQuery(db *gorm.DB) *gorm.DB {
	return db.Where("user_id IN ?", a.List)
}
func (a OrderWithList) BindFirst(db *gorm.DB) *gorm.DB {
	return db
}
func (a OrderWithList) BindCount(db *gorm.DB) *gorm.DB {
	return db
}
func (a OrderWithList) Sortable() map[string]bool {
	return map[string]bool{}
}
