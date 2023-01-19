package storage

import (
	"code.cryptopower.dev/mgmt-ng/be/payment"
	"fmt"
	"time"
)

const UserFieldUName = "user_name"
const UserFieldId = "id"

type UserRole int

const (
	UserRoleNone UserRole = iota
	UserRoleAdmin
)

type UserStorage interface {
	CreateUser(user *User) error
	UpdateUser(user *User) error
	QueryUser(field string, val interface{}) (*User, error)
}

type User struct {
	Id             uint64 `gorm:"primarykey"`
	UserName       string `gorm:"index:users_user_name_idx,unique"`
	DisplayName    string
	PasswordHash   string `json:"-"`
	Email          string
	PaymentType    payment.Type
	PaymentAddress string
	Role           UserRole
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastSeen       time.Time
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
