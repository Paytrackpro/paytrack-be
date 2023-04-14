package portal

import (
	"fmt"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
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
	Amount          float64                 `json:"amount"`
	Description     string                  `json:"description"`
	Details         []storage.PaymentDetail `json:"details"`
	PaymentMethod   utils.Method            `json:"paymentMethod"`
	PaymentAddress  string                  `json:"paymentAddress"`
	Status          storage.PaymentStatus   `json:"status"`
	TxId            string                  `json:"txId"`
	IsDraft         bool                    `json:"isDraft"`
	Token           string                  `json:"token"`
}

type PaymentConfirm struct {
	Id             uint64       `validate:"required" json:"id"`
	TxId           string       `json:"txId"`
	Token          string       `json:"token"`
	PaymentMethod  utils.Method `validate:"required" json:"paymentMethod"`
	PaymentAddress string       `validate:"required" json:"paymentAddress"`
}

func (p *PaymentRequest) calculateAmount() (float64, error) {
	var amount float64
	for i, detail := range p.Details {
		if detail.Quantity > 0 {
			var price = p.HourlyRate
			if detail.Price > 0 {
				price = detail.Price
			}
			cost := detail.Quantity * price
			if cost != detail.Cost {
				return 0, fmt.Errorf("payment detail amount is incorrect at line %d", i+1)
			}
			if detail.Cost <= 0 {
				return 0, fmt.Errorf("payment detail cost must be greater than 0 at line %d", i+1)
			}
		}
		amount += detail.Cost
	}
	return amount, nil
}

func (p *PaymentRequest) Payment(userId uint64, payment *storage.Payment, isHaveApprover bool) error {
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
	if payment.SenderId == userId {
		payment.HourlyRate = p.HourlyRate
		payment.Details = p.Details
		payment.PaymentMethod = p.PaymentMethod
		payment.PaymentAddress = p.PaymentAddress
		payment.PaymentSettings = p.PaymentSettings

		if len(p.Details) > 0 {
			amount, err := p.calculateAmount()
			if err != nil {
				return err
			}
			payment.Amount = amount
			payment.Description = ""
			payment.HourlyRate = p.HourlyRate
			payment.Details = p.Details
		} else {
			payment.HourlyRate = 0
			payment.Details = nil
			payment.Description = p.Description
			payment.Amount = p.Amount
		}
		if payment.Amount == 0 {
			return fmt.Errorf("amount must not be zero")
		}
		// allow the sender edit the receiver
		if p.ContactMethod == storage.PaymentTypeInternal {
			payment.ExternalEmail = ""
			payment.ReceiverId = p.ReceiverId
		} else {
			payment.ReceiverId = 0
			payment.ExternalEmail = p.ExternalEmail
		}
	}
	// sender sent the request to the recipient
	if userId == payment.SenderId && p.Status == storage.PaymentStatusSent {
		if isHaveApprover {
			payment.Approvers = make(storage.Approvers, 0)
		}
		// If the payment is rejected and user update and re-send with new status, then we need to update the status to sent
		if payment.Status == storage.PaymentStatusCreated || payment.Status == storage.PaymentStatusRejected {
			payment.Status = storage.PaymentStatusSent
			payment.SentAt = time.Now()
		}
	}

	// recipient update status and txId
	if userId == payment.ReceiverId && p.Status != storage.PaymentStatusCreated {
		// allow recipient update status to sent or confirmed
		if p.Status == storage.PaymentStatusSent || p.Status == storage.PaymentStatusConfirmed {
			payment.Status = p.Status
		}
		payment.TxId = p.TxId
	}
	return nil
}

func (p *PaymentConfirm) Process(payment *storage.Payment) {
	payment.TxId = p.TxId
	payment.PaidAt = time.Now()
	payment.Status = storage.PaymentStatusPaid
}

type PaymentRequestRate struct {
	Id             uint64       `json:"id" validate:"required"`
	Token          string       `json:"token"`
	PaymentMethod  utils.Method `json:"paymentMethod"`
	PaymentAddress string       `json:"paymentAddress"`
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

type PaymentReject struct {
	Id              uint64 `json:"id" validate:"required"`
	Token           string `json:"token"`
	RejectionReason string `json:"rejectionReason"`
}

type BulkPaymentBTC struct {
	ID             int          `json:"id"`
	Rate           float64      `json:"rate"`
	ConvertTime    int64        `json:"convertTime"`
	PaymentAddress string       `json:"paymentAddress"`
	PaymentMethod  utils.Method `json:"paymentMethod"`
	PaymentToken   string       `json:"token"`
}

type BulkPaidRequests struct {
	TxId        string           `json:"txId"`
	PaymentList []BulkPaymentBTC `json:"paymentList"`
}

type BulkPaidRequest struct {
	PaymentIds []int  `json:"paymentIds"`
	TXID       string `json:"txid"`
}

type GetRateRequest struct {
	Symbol utils.Method `json:"symbol"`
}

type GetRateResponse struct {
	Rate        float64 `json:"rate"`
	ConvertTime int64   `json:"convertTime"`
}
