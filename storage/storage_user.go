package storage

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/payment"
	"code.cryptopower.dev/mgmt-ng/be/utils"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"gorm.io/gorm"
)

const UserFieldUName = "user_name"
const UserFieldId = "id"

type UserStorage interface {
	CheckDuplicate(user *User) error
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
	Id              uint64          `json:"id" gorm:"primarykey"`
	UserName        string          `json:"userName" gorm:"index:users_user_name_idx,unique"`
	DisplayName     string          `json:"displayName"`
	PasswordHash    string          `json:"-"`
	Email           string          `json:"email"`
	PaymentSettings PaymentSettings `json:"paymentSettings" gorm:"type:jsonb"`
	Role            utils.UserRole  `json:"role"`
	CreatedAt       time.Time       `json:"createdAt"`
	UpdatedAt       time.Time       `json:"updatedAt"`
	LastSeen        time.Time       `json:"lastSeen"`
	Secret          string          `json:"-"`
	Otp             bool            `json:"otp"`
	credentials     []webauthn.Credential
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
		"createdAt": true,
		"lastSeen":  true,
	}
}

func (u User) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

func (u User) WebAuthnName() string {
	return u.UserName
}

// WebAuthnDisplayName returns the user's display name
func (u User) WebAuthnDisplayName() string {
	return u.DisplayName
}

func randomUint64() uint64 {
	buf := make([]byte, 8)
	rand.Read(buf)
	return binary.LittleEndian.Uint64(buf)
}

// WebAuthnID returns the user's ID
func (u User) WebAuthnID() []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(buf, uint64(u.Id))
	return buf
}

func (u User) WebAuthnIcon() string {
	return ""
}

// AddCredential associates the credential to the user
func (u *User) AddCredential(cred webauthn.Credential) {
	u.credentials = append(u.credentials, cred)
}

// CredentialExcludeList returns a CredentialDescriptor array filled
// with all the user's credentials
func (u User) CredentialExcludeList() []protocol.CredentialDescriptor {

	credentialExcludeList := []protocol.CredentialDescriptor{}
	for _, cred := range u.credentials {
		descriptor := protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: cred.ID,
		}
		credentialExcludeList = append(credentialExcludeList, descriptor)
	}

	return credentialExcludeList
}
