package portal

import (
	"code.cryptopower.dev/mgmt-ng/be/payment"
)

type UpdateUserRequest struct {
	DisplayName    string         `json:"displayName"`
	Password       string         `json:"password"`
	Email          string         `validate:"omitempty,email" json:"email"`
	PaymentType    payment.Method `json:"paymentType"`
	PaymentAddress string         `json:"paymentAddress"`
	UserId         int            `json:"userId"`
	Otp            bool           `json:"otp"`
}
