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
	DefaultPayment payment.Type
	PaymentAddress string
}

type LoginForm struct {
	UserName string `validate:"required,alphanum,gte=4,lte=32"`
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
		PaymentType:  f.DefaultPayment,
	}
	if user.PaymentType != payment.PaymentTypeNotSet {
		user.PaymentAddress = f.PaymentAddress
	}
	return &user, nil
}
