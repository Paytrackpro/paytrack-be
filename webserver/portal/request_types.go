package portal

import (
	"code.cryptopower.dev/mgmt-ng/be/utils"
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
	PaymentType    utils.PaymentType
	PaymentAddress string
	UserId         int
}
