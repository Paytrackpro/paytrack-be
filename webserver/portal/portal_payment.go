package portal

import (
	"code.cryptopower.dev/mgmt-ng/be/payment"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"time"
)

type PaymentRequest struct {
	// Sender is the person who will pay for the payment
	SenderId       uint64                 `validate:"required_if=ContactMethod 0" json:"senderId"`
	SenderEmail    string                 `validate:"required_if=ContactMethod 1,omitempty,email" json:"senderEmail"`
	ContactMethod  storage.PaymentContact `json:"contactMethod"`
	Amount         float64                `validate:"required" json:"amount"`
	Description    string                 `validate:"required" json:"description"`
	PaymentMethod  payment.Method         `validate:"required" json:"paymentMethod"`
	PaymentAddress string                 `validate:"required" json:"paymentAddress"`
}

type PaymentConfirm struct {
	Id    uint64 `validate:"required" json:"id"`
	TxId  string `json:"txId"`
	Token string `json:"token"`
}

func (p *PaymentRequest) Payment(requesterId uint64) storage.Payment {
	var payment = storage.Payment{
		ContactMethod:  p.ContactMethod,
		RequesterId:    requesterId,
		Amount:         p.Amount,
		Description:    p.Description,
		PaymentMethod:  p.PaymentMethod,
		PaymentAddress: p.PaymentAddress,
		Status:         storage.PaymentStatusCreated,
	}
	if p.ContactMethod == storage.PaymentTypeInternal {
		payment.SenderId = p.SenderId
	}
	if p.ContactMethod == storage.PaymentTypeEmail {
		payment.SenderEmail = p.SenderEmail
	}
	return payment
}

func (p *PaymentConfirm) Process(payment *storage.Payment) {
	payment.TxId = p.TxId
	payment.PaidAt = time.Now()
	payment.Status = storage.PaymentStatusPaid
}

type PaymentRequestRate struct {
	Id    uint64 `json:"id" validate:"required"`
	Token string `json:"token"`
}
