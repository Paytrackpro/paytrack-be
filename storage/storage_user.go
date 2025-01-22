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

type AuthType int

const (
	AuthLocalUsernamePassword AuthType = iota
	AuthMicroservicePasskey
)

type UserStorage interface {
	CheckDuplicate(user *User) error
	CreateUser(user *User) error
	CreateUserTimer(userTimer *UserTimer) error
	UpdateUserTimer(userTimer *UserTimer) error
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
	ShowApproved          bool            `json:"showApproved"`
	Secret                string          `json:"-"`
	Otp                   bool            `json:"otp"`
	Locked                bool            `json:"locked"`
	ShowDraftForRecipient bool            `json:"showDraftForRecipient"`
	AuthType              int             `json:"authType"`
	ShowDateOnInvoiceLine bool            `json:"showDateOnInvoiceLine"`
}

type AuthClaims struct {
	Id          int64  `json:"id"`
	Username    string `json:"username"`
	Expire      int64  `json:"expire"`
	Role        int    `json:"role"`
	Createdt    int64  `json:"createdt"`
	LastLogindt int64  `json:"lastLogindt"`
}

type UserWorkingDisplay struct {
	User
	Working bool `json:"working"`
	Pausing bool `json:"pausing"`
}

type UserTimer struct {
	Id          uint64        `json:"id" gorm:"primarykey"`
	UserId      uint64        `json:"userId"`
	Start       time.Time     `json:"start"`
	Stop        time.Time     `json:"stop"`
	PauseState  PauseStatuses `json:"pauseState" gorm:"type:jsonb"`
	Duration    uint64        `json:"duration"`
	Fininshed   bool          `json:"fininshed"`
	Pausing     bool          `json:"pausing"`
	ProjectId   uint64        `json:"projectId"`
	Description string        `json:"description"`
}

type UserTimerSockerData struct {
	UserId  uint64
	Working bool
	Pausing bool
}

type PauseStatuses []PauseStatus

type PauseStatus struct {
	Start time.Time
	Stop  time.Time
}

// Value Marshal
func (a PauseStatuses) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan Unmarshal
func (a *PauseStatuses) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

func (p *psql) CreateUserTimer(userTimer *UserTimer) error {
	return p.db.Create(userTimer).Error
}

func (p *psql) UpdateUserTimer(userTimer *UserTimer) error {
	return p.db.Save(userTimer).Error
}

func (UserTimer) TableName() string {
	return "user_timer"
}

func (p *psql) QueryUserTimerWithList(field string, val interface{}) ([]UserTimer, error) {
	var user []UserTimer
	var err = p.db.Where(fmt.Sprintf("%s IN ?", field), val).Find(&user).Error
	return user, err
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
	LastSeen  string
}

type AdminReportFilter struct {
	Sort
	StartDate time.Time
	EndDate   time.Time
	UserName  string // DungPA: Task3
}

type AdminReportFilterUserDetail struct {
	Sort
	Sent        bool
	Received    bool
	Paid        bool
	HasBeenPaid bool
	UserName    string
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
	if !utils.IsEmpty(f.LastSeen) && f.LastSeen == "3MOTH" {
		threeMonthsAgo := time.Now().AddDate(0, -3, 0)
		db = db.Where("last_seen >= ?", threeMonthsAgo)
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
