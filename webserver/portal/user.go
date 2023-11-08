package portal

import (
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type RegisterForm struct {
	UserName       string `validate:"required,alphanum,gte=4,lte=32"`
	DisplayName    string
	Password       string `validate:"required"`
	Email          string `validate:"omitempty,email"`
	DefaultPayment utils.Method
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
		DisplayName:  f.DisplayName,
	}
	if f.DefaultPayment != utils.PaymentTypeNotSet {
		user.PaymentSettings = []storage.PaymentSetting{
			{Type: f.DefaultPayment, Address: f.PaymentAddress},
		}
	}
	return &user, nil
}

type UpdateUserRequest struct {
	UserName              string                   `json:"userName"`
	DisplayName           string                   `json:"displayName"`
	ShopName              string                   `json:"shopName"`
	Password              string                   `json:"password"`
	Email                 string                   `validate:"omitempty,email" json:"email"`
	PaymentType           utils.Method             `json:"paymentType"`
	PaymentAddress        string                   `json:"paymentAddress"`
	UserId                int                      `json:"userId"`
	Otp                   bool                     `json:"otp"`
	PaymentSettings       []storage.PaymentSetting `json:"paymentSettings"`
	HourlyLaborRate       float64                  `json:"hourlyLaborRate"`
	Locked                bool                     `json:"locked"`
	ShowMonthlyReport     bool                     `json:"showMonthlyReport"`
	ShowDraftForRecipient bool                     `json:"showDraftForRecipient"`
	ShowDateOnInvoiceLine bool                     `json:"showDateOnInvoiceLine"`
	Role                  utils.UserRole           `json:"role"`
}

type UserWithList struct {
	List []uint64
}

func (a UserWithList) RequestedSort() string {
	return ""
}
func (a UserWithList) BindQuery(db *gorm.DB) *gorm.DB {
	return db.Where("id IN ?", a.List)
}
func (a UserWithList) BindFirst(db *gorm.DB) *gorm.DB {
	return db
}
func (a UserWithList) BindCount(db *gorm.DB) *gorm.DB {
	return db
}
func (a UserWithList) Sortable() map[string]bool {
	return map[string]bool{}
}

type Approvers struct {
	Id         uint64
	ApproverId uint64
	SenderId   uint64
}

func (a Approvers) RequestedSort() string {
	return ""
}
func (a Approvers) BindQuery(db *gorm.DB) *gorm.DB {
	return db.Where("recipient_id = ?", a.Id)
}
func (a Approvers) BindFirst(db *gorm.DB) *gorm.DB {
	return db
}
func (a Approvers) BindCount(db *gorm.DB) *gorm.DB {
	return db
}
func (a Approvers) Sortable() map[string]bool {
	return map[string]bool{}
}

type ChangePasswordRequest struct {
	Password    string `json:"password"`
	OldPassword string `json:"oldPassword"`
	Otp         string `json:"otp"`
}
