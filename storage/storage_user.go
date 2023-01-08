package storage

import (
	"fmt"
	"time"
)

const UserFieldUName = "user_name"
const UserFieldId = "id"

type UserStorage interface {
	CreateUser(user *User) error
	UpdateUser(user *User) error
	QueryUser(field, val string) (*User, error)
	ListUser() ([]User, error)
}

type User struct {
	Id           string
	UserName     string `gorm:"index:users_user_name_idx,unique"`
	PasswordHash string `json:"-"`
	Email        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (User) TableName() string {
	return "users"
}

func (p *psql) CreateUser(user *User) error {
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	return p.db.Create(user).Error
}

func (p *psql) UpdateUser(user *User) error {
	user.UpdatedAt = time.Now()
	return p.db.Save(user).Error
}

func (p *psql) QueryUser(field, val string) (*User, error) {
	var user User
	var err = p.db.Where(fmt.Sprintf("%s = ?", field), val).First(&user).Error
	return &user, err
}

func (p *psql) ListUser() ([]User, error) {
	return nil, nil
}
