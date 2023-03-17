package portal

import (
	"fmt"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/payment"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"gorm.io/gorm"
)

type PaymentRequest struct {
	// Sender is the person who will pay for the payment
	SenderId   uint64 `validate:"required_if=ContactMethod 0" json:"senderId"`
	ReceiverId uint64 `json:"receiverId"`
	// ExternalEmail is the field to send the payment to the person who does not have an account yet
	ExternalEmail   string                  `validate:"required_if=ContactMethod 1,omitempty,email" json:"externalEmail"`
	ContactMethod   storage.PaymentContact  `json:"contactMethod"`
	HourlyRate      float64                 `json:"hourlyRate"`
	PaymentSettings storage.PaymentSettings `json:"paymentSettings" gorm:"type:jsonb"`
	Details         []storage.PaymentDetail `json:"details"`
	PaymentMethod   payment.Method          `json:"paymentMethod"`
	PaymentAddress  string                  `json:"paymentAddress"`
	Status          storage.PaymentStatus   `json:"status"`
	TxId            string                  `json:"txId"`
	IsDraft         bool                    `json:"isDraft"`
	Token           string                  `json:"token"`
}

type PaymentConfirm struct {
	Id             uint64         `validate:"required" json:"id"`
	TxId           string         `json:"txId"`
	Token          string         `json:"token"`
	PaymentMethod  payment.Method `validate:"required" json:"paymentMethod"`
	PaymentAddress string         `validate:"required" json:"paymentAddress"`
}

func (p *PaymentRequest) Payment(userId uint64, payment *storage.Payment) error {
	if payment.Id == 0 {
		payment.CreatorId = userId
		payment.SenderId = userId
		payment.ReceiverId = p.ReceiverId
		payment.ExternalEmail = p.ExternalEmail
		payment.ContactMethod = p.ContactMethod
	}
	if !(userId == p.SenderId || userId == p.ReceiverId) {
		return fmt.Errorf("the sender or receiver must be you")
	}
	// allow the sender edit the receiver
	if payment.Id > 0 && payment.SenderId == userId {
		if p.ContactMethod == storage.PaymentTypeInternal {
			payment.ExternalEmail = ""
			payment.ReceiverId = p.ReceiverId
		} else {
			payment.ReceiverId = 0
			payment.ExternalEmail = p.ExternalEmail
		}
	}
	payment.HourlyRate = p.HourlyRate
	payment.Details = p.Details
	payment.PaymentMethod = p.PaymentMethod
	payment.PaymentAddress = p.PaymentAddress
	payment.PaymentSettings = p.PaymentSettings

	// sender sent the request to the recipient
	if userId == payment.SenderId && payment.Status == storage.PaymentStatusCreated && p.Status == storage.PaymentStatusSent {
		payment.Status = storage.PaymentStatusSent
		payment.SentAt = time.Now()
	}
	// recipient update status and txId
	if userId == payment.ReceiverId && p.Status != storage.PaymentStatusCreated {
		// allow recipient update status to sent or confirmed
		if p.Status == storage.PaymentStatusSent || p.Status == storage.PaymentStatusConfirmed {
			payment.Status = p.Status
		}
		payment.TxId = p.TxId
	}
	// calculate amount
	var amount float64
	for i, detail := range p.Details {
		if detail.Hours > 0 {
			cost := detail.Hours * payment.HourlyRate
			if cost != detail.Cost {
				return fmt.Errorf("payment detail is wrong amount at line %d", i+1)
			}
			if detail.Cost == 0 {
				return fmt.Errorf("payment detail is 0 cost at line %d", i+1)
			}
		}
		amount += detail.Cost
	}
	payment.Amount = amount
	return nil
}

func (p *PaymentConfirm) Process(payment *storage.Payment) {
	payment.TxId = p.TxId
	payment.PaidAt = time.Now()
	payment.Status = storage.PaymentStatusPaid
}

type PaymentRequestRate struct {
	Id             uint64         `json:"id" validate:"required"`
	Token          string         `json:"token"`
	PaymentMethod  payment.Method `json:"paymentMethod"`
	PaymentAddress string         `json:"paymentAddress"`
}

type ListPaymentSettingRequest struct {
	Id   uint64
	List []ApproversSettingRequest `json:"list"`
}

type ApproversSettingRequest struct {
	ApproverIds []uint64 `json:"approverIds"`
	SendUserId  uint64   `json:"sendUserId"`
}

func (p *ListPaymentSettingRequest) MakeApproverSetting(id uint64, userMap map[uint64]storage.User) []storage.ApproverSettings {

	sets := make([]storage.ApproverSettings, 0)
	for _, setting := range p.List {
		for _, v := range setting.ApproverIds {
			sets = append(sets, storage.ApproverSettings{
				ApproverId:   v,
				SendUserId:   setting.SendUserId,
				RecipientId:  id,
				ApproverName: userMap[v].UserName,
				SendUserName: userMap[setting.SendUserId].UserName,
			})
		}
	}
	return sets
}

func (a ListPaymentSettingRequest) BindQueryDelete(db *gorm.DB) *gorm.DB {
	return db.Where("recipient_id", a.Id)
}
