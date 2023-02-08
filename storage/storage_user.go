package storage

import (
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

type User struct {
	Id             uint64         `json:"id" gorm:"primarykey"`
	UserName       string         `json:"userName" gorm:"index:users_user_name_idx,unique"`
	DisplayName    string         `json:"displayName"`
	PasswordHash   string         `json:"-"`
	Email          string         `json:"email"`
	PaymentType    payment.Method `json:"paymentType"`
	PaymentAddress string         `json:"paymentAddress"`
	Role           utils.UserRole `json:"role"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	LastSeen       time.Time      `json:"lastSeen"`
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
	if !utils.IsEmpty(f.KeySearch) {
		keySearch := fmt.Sprintf("%%%s%%", strings.TrimSpace(f.KeySearch))
		db = db.Where("user_name LIKE ?", keySearch)
	}
	return db
}

func (f *UserFilter) Sortable() map[string]bool {
	return map[string]bool{
		"createdAt": true,
		"lastSeen":  true,
	}
}
