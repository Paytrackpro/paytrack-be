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

func (p *PaymentRequest) Payment(creatorId uint64, payment *storage.Payment) error {
	if !(creatorId == p.SenderId || creatorId == p.ReceiverId) {
		return fmt.Errorf("the sender or receiver must be you")
	}
	if payment.Id == 0 {
		payment.CreatorId = creatorId
	}
	payment.ContactMethod = p.ContactMethod
	payment.HourlyRate = p.HourlyRate
	payment.Details = p.Details
	payment.PaymentMethod = p.PaymentMethod
	payment.PaymentAddress = p.PaymentAddress
	payment.SenderId = p.SenderId
	payment.ReceiverId = p.ReceiverId
	payment.PaymentSettings = p.PaymentSettings
	if payment.Id == 0 || payment.CreatorId == creatorId {
		if p.ContactMethod == storage.PaymentTypeInternal {
			payment.ExternalEmail = ""
		}
		if p.ContactMethod == storage.PaymentTypeEmail {
			if p.SenderId != 0 && p.ReceiverId != 0 {
				return fmt.Errorf("invalid method")
			}
			payment.ExternalEmail = p.ExternalEmail
		}
	}
	if !p.IsDraft && payment.Status == storage.PaymentStatusCreated {
		payment.Status = storage.PaymentStatusSent
		payment.SentAt = time.Now()
	}
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
	ApproverId  uint64 `json:"approverId"`
	SendUserId  uint64 `json:"sendUserId"`
	RecipientId uint64 `json:"recipientId"`
}

// func (p *ListPaymentSettingRequest) MakeApproverSetting() []storage.ApproverSettings {
// 	sets := make([]storage.ApproverSettings, 0)
// 	for _, setting := range p.List {
// 		sets = append(sets, storage.ApproverSettings{
// 			ApproverId:  setting.ApproverId,
// 			SendUserId:  setting.SendUserId,
// 			RecipientId: setting.RecipientId,
// 		})
// 	}
// 	return sets
// }

func (a *ListPaymentSettingRequest) BindQueryDelete(db *gorm.DB) *gorm.DB {
	return db.Where("recipient_id", a.Id)
}
