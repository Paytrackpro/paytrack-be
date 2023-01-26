package portal

import (
	"code.cryptopower.dev/mgmt-ng/be/payment"
	"code.cryptopower.dev/mgmt-ng/be/storage"
	"time"
)

type PaymentRequest struct {
	// Sender is the person who will pay for the payment
	SenderId       uint64 `validate:"required_if=ContactMethod 0"`
	SenderEmail    string `validate:"required_if=ContactMethod 1"`
	ContactMethod  storage.PaymentContact
	Amount         float64        `validate:"required"`
	Description    string         `validate:"required"`
	PaymentMethod  payment.Method `validate:"required"`
	PaymentAddress string         `validate:"required"`
}

type PaymentConfirm struct {
	Id   uint64 `validate:"required"`
	TxId string
}

func (p *PaymentRequest) Payment(requesterId uint64) storage.Payment {
	return storage.Payment{
		RequesterId:    requesterId,
		SenderId:       p.SenderId,
		SenderEmail:    p.SenderEmail,
		Amount:         p.Amount,
		Description:    p.Description,
		PaymentMethod:  p.PaymentMethod,
		PaymentAddress: p.PaymentAddress,
		Status:         storage.PaymentStatusCreated,
	}
}

func (p *PaymentConfirm) Process(payment *storage.Payment) {
	payment.TxId = p.TxId
	payment.PaidAt = time.Now()
	payment.Status = storage.PaymentStatusPaid
}
