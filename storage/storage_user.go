package storage

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/utils"
	"gorm.io/gorm"
)

const UserFieldUName = "user_name"
const UserFieldId = "id"
const RecipientId = "recipient_id"

type UserStorage interface {
	CheckDuplicate(user *User) error
	CreateUser(user *User) error
	UpdateUser(user *User) error
	QueryUser(field string, val interface{}) (*User, error)
	QueryUserWithList(field string, val interface{}) ([]User, error)
}

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
	Id                    uint64          `json:"id" gorm:"primarykey"`
	UserName              string          `json:"userName" gorm:"index:users_user_name_idx,unique"`
	DisplayName           string          `json:"displayName"`
	PasswordHash          string          `json:"-"`
	Email                 string          `json:"email"`
	HourlyLaborRate       float64         `json:"hourlyLaborRate"`
	PaymentSettings       PaymentSettings `json:"paymentSettings" gorm:"type:jsonb"`
	Role                  utils.UserRole  `json:"role"`
	CreatedAt             time.Time       `json:"createdAt"`
	UpdatedAt             time.Time       `json:"updatedAt"`
	LastSeen              time.Time       `json:"lastSeen" gorm:"default:current_timestamp"`
	HidePaid              bool            `json:"hidePaid"`
	Secret                string          `json:"-"`
	Otp                   bool            `json:"otp"`
	Locked                bool            `json:"locked"`
	ShowDraftForRecipient bool            `json:"showDraftForRecipient"`
	ShowDateOnInvoiceLine bool            `json:"showDateOnInvoiceLine"`
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

func (p *psql) CheckDuplicate(user *User) error {
	if user.Email == "" {
		return nil
	}
	var oldUser User
	var err = p.db.Where("email", user.Email).Not("id", user.Id).First(&oldUser).Error
	if err == nil {
		return fmt.Errorf("the email is already taken")
	}
	return nil
}

func (p *psql) QueryUser(field string, val interface{}) (*User, error) {
	var user User
	var err = p.db.Where(fmt.Sprintf("%s = ?", field), val).First(&user).Error
	return &user, err
}

func (p *psql) QueryUserWithList(field string, val interface{}) ([]User, error) {
	var user []User
	var err = p.db.Where(fmt.Sprintf("%s IN ?", field), val).Find(&user).Error
	return user, err
}

type UserFilter struct {
	Sort
	KeySearch string
	Email     string
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
	if len(f.Email) > 0 {
		db = db.Where("email", f.Email)
	}
	return db
}

func (f *UserFilter) Sortable() map[string]bool {
	return map[string]bool{
		"userName":    true,
		"displayName": true,
		"email":       true,
		"createdAt":   true,
		"lastSeen":    true,
	}
}
