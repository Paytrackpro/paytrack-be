package portal

import (
	"code.cryptopower.dev/mgmt-ng/be/payment"
)

type ListUserRequest struct {
	SortType  int    `schema:"sortType"`
	Sort      int    `schema:"sort"`
	KeySearch string `schema:"keySearch"`
	Limit     int    `schema:"limit"`
	Offset    int    `schema:"offset"`
}

type UpdateUserRequest struct {
	Password       string
	Email          string
	PaymentType    payment.Method
	PaymentAddress string
	UserId         int
}
