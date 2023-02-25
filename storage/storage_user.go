package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/payment"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"gorm.io/gorm"
)

const UserFieldUName = "user_name"
const UserFieldId = "id"

type UserStorage interface {
	CreateUser(user *User) error
	UpdateUser(user *User) error
	QueryUser(field string, val interface{}) (*User, error)
}

type PaymentSetting struct {
	Type      payment.Method `json:"type"`
	Address   string         `json:"address"`
	IsDefault bool           `json:"isDefault"`
}

type PaymentSettings []PaymentSetting

// Value Marshal
func (a PaymentSettings) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan Unmarshal
func (a *PaymentSettings) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

type User struct {
	Id              uint64           `json:"id" gorm:"primarykey"`
	UserName        string           `json:"userName" gorm:"index:users_user_name_idx,unique"`
	DisplayName     string           `json:"displayName"`
	PasswordHash    string           `json:"-"`
	Email           string           `json:"email"`
	PaymentSettings []PaymentSetting `json:"paymentSettings" gorm:"type:jsonb"`
	Role            utils.UserRole   `json:"role"`
	CreatedAt       time.Time        `json:"createdAt"`
	UpdatedAt       time.Time        `json:"updatedAt"`
	LastSeen        time.Time        `json:"lastSeen"`
	Secret          string           `json:"-"`
	Otp             bool             `json:"otp"`
}

func (User) TableName() string {
	return "users"
}

func (p *psql) CreateUser(user *User) error {
	return p.db.Create(user).Error
}

func (p *psql) UpdateUser(user *User) error {
	return p.db.Save(user).Error
}

func (p *psql) QueryUser(field string, val interface{}) (*User, error) {
	var user User
	var err = p.db.Where(fmt.Sprintf("%s = ?", field), val).First(&user).Error
	return &user, err
}

type UserFilter struct {
	Sort
	KeySearch string
}

func (f *UserFilter) BindQuery(db *gorm.DB) *gorm.DB {
	db = f.Sort.BindQuery(db)
	return f.BindCount(db)
}

func (f *UserFilter) BindCount(db *gorm.DB) *gorm.DB {
	if !utils.IsEmpty(f.KeySearch) {
		keySearch := fmt.Sprintf("%%%s%%", strings.TrimSpace(f.KeySearch))
		db = db.Where("user_name LIKE ?", keySearch)
	}
	return db
}

func (f *UserFilter) BindFirst(db *gorm.DB) *gorm.DB {
	return db
}

func (f *UserFilter) Sortable() map[string]bool {
	return map[string]bool{
		"createdAt": true,
		"lastSeen":  true,
	}
}
