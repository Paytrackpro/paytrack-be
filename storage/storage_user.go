package storage

import (
	"fmt"
	"strings"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/models"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"gorm.io/gorm"
)

const UserFieldUName = "user_name"
const UserFieldId = "id"

type UserStorage interface {
	CreateUser(user *User) error
	UpdateUser(user *User) error
	QueryUser(field string, val interface{}) (*User, error)
	GetListUser(filter models.UserFilter) ([]User, error)
}

type User struct {
	Id             uint64            `json:"id" gorm:"primarykey"`
	UserName       string            `json:"user_name" gorm:"index:users_user_name_idx,unique"`
	PasswordHash   string            `json:"-"`
	Email          string            `json:"email"`
	PaymentType    utils.PaymentType `json:"payment_type"`
	PaymentAddress string            `json:"payment_address"`
	Status         utils.UserStatus  `gorm:"default:1" json:"status"`
	Role           utils.UserRole    `json:"role"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	LastSeen       time.Time         `json:"last_seen"`
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

func (p *psql) GetListUser(filter models.UserFilter) ([]User, error) {
	builder := p.db
	if !utils.IsEmpty(filter.KeySearch) {
		keySearch := fmt.Sprintf("%%%s%%", strings.TrimSpace(filter.KeySearch))
		builder = builder.Where("user_name LIKE ?", keySearch)
	}

	if !utils.IsEmpty(filter.SortType) {
		s := "desc"
		if filter.Sort == utils.SortASC {
			s = "asc"
		}

		if filter.SortType == utils.SortByCreated {
			builder = builder.Order("created_at " + s)
		}

		if filter.SortType == utils.SortByCreated {
			builder = builder.Order("last_seen " + s)
		}
	}

	user := make([]User, 0)
	if err := builder.Limit(filter.Limit).Offset(filter.Offset).Find(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return user, nil
		}
		return nil, err
	}

	return user, nil
}
