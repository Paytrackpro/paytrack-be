package portal

import (
	"code.cryptopower.dev/mgmt-ng/be/payment"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"golang.org/x/crypto/bcrypt"
)

type RegisterForm struct {
	UserName       string `validate:"required,alphanum,gte=4,lte=32"`
	Password       string `validate:"required"`
	Email          string `validate:"omitempty,email"`
	DefaultPayment payment.Method
	PaymentAddress string
}

type LoginForm struct {
	UserName string `validate:"required,alphanum,gte=4,lte=32"`
	Password string `validate:"required"`
	IsOtp    bool   `validate:"required"`
	Otp      string
}

type OtpForm struct {
	Otp       string `validate:"required"`
	Password  string `validate:"required"`
	FirstTime bool
}

type GenerateQRForm struct {
	Password string `validate:"required"`
}

func (f RegisterForm) User() (*storage.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(f.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	var user = storage.User{
		UserName:     f.UserName,
		PasswordHash: string(hash),
		Email:        f.Email,
	}
	if f.DefaultPayment != payment.PaymentTypeNotSet {
		user.PaymentSettings = []storage.PaymentSetting{
			{Type: f.DefaultPayment, Address: f.PaymentAddress},
		}
	}
	return &user, nil
}

type UpdateUserRequest struct {
	DisplayName     string                   `json:"displayName"`
	Password        string                   `json:"password"`
	Email           string                   `validate:"omitempty,email" json:"email"`
	PaymentType     payment.Method           `json:"paymentType"`
	PaymentAddress  string                   `json:"paymentAddress"`
	UserId          int                      `json:"userId"`
	Otp             bool                     `json:"otp"`
	PaymentSettings []storage.PaymentSetting `json:"paymentSettings"`
}
