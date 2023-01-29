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

type UserRole int

const (
	UserRoleNone UserRole = iota
	UserRoleAdmin
)

type UserStorage interface {
	CreateUser(user *User) error
	UpdateUser(user *User) error
	QueryUser(field string, val interface{}) (*User, error)
	GetListUser(sortType, sort, limit, offset int, keySearch string) ([]User, error)
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

func (p *psql) GetListUser(sortType, sort, limit, offset int, keySearch string) ([]User, error) {
	builder := p.db
	if !utils.IsEmpty(keySearch) {
		keySearch := fmt.Sprintf("%%%s%%", strings.TrimSpace(keySearch))
		builder = builder.Where("user_name LIKE ?", keySearch)
	}

	if !utils.IsEmpty(sortType) {
		s := "desc"
		if sort == utils.SortASC {
			s = "asc"
		}

		if sortType == utils.SortByCreated {
			builder = builder.Order("created_at " + s)
		}

		if sortType == utils.SortByCreated {
			builder = builder.Order("last_seen " + s)
		}
	}

	user := make([]User, 0)
	if err := builder.Limit(limit).Offset(offset).Find(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return user, nil
		}
		return nil, err
	}

	return user, nil
}
