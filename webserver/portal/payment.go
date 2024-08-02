package portal

import (
	"time"

	"code.cryptopower.dev/mgmt-ng/be/storage"
	"code.cryptopower.dev/mgmt-ng/be/utils"
)

type PaymentRequest struct {
	// Sender is the person who will pay for the payment
	SenderId   uint64 `validate:"required_if=ContactMethod 0" json:"senderId"`
	ReceiverId uint64 `json:"receiverId"`

	// ExternalEmail is the field to send the payment to the person who does not have an account yet
	ExternalEmail         string                  `validate:"required_if=ContactMethod 1,omitempty,email" json:"externalEmail"`
	ContactMethod         storage.PaymentContact  `json:"contactMethod"`
	HourlyRate            float64                 `json:"hourlyRate"`
	PaymentSettings       storage.PaymentSettings `json:"paymentSettings" gorm:"type:jsonb"`
	Amount                float64                 `json:"amount"`
	Description           string                  `json:"description"`
	Details               []storage.PaymentDetail `json:"details"`
	PaymentMethod         utils.Method            `json:"paymentMethod"`
	PaymentAddress        string                  `json:"paymentAddress"`
	Status                storage.PaymentStatus   `json:"status"`
	TxId                  string                  `json:"txId"`
	Token                 string                  `json:"token"`
	ReceiptImg            string                  `json:"receiptImg"`
	ShowDraftRecipient    bool                    `json:"showDraftRecipient"`
	ShowDateOnInvoiceLine bool                    `json:"showDateOnInvoiceLine"`
	ShowProjectOnInvoice  bool                    `json:"showProjectOnInvoice"`
	ProjectId             uint64                  `json:"projectId"`
	ProjectName           string                  `json:"projectName"`
}

type PaymentBTCPayInvoice struct {
	Id uint64 `json:"id"`
}

type PaymentConfirm struct {
	Id             uint64       `validate:"required" json:"id"`
	TxId           string       `json:"txId"`
	Token          string       `json:"token"`
	ConvertRate    float64      `json:"convertRate"`
	ConvertTime    time.Time    `json:"convertTime"`
	ExpectedAmount float64      `json:"expectedAmount"`
	PaymentMethod  utils.Method `validate:"required" json:"paymentMethod"`
	PaymentAddress string       `validate:"required" json:"paymentAddress"`
}

func (p *PaymentConfirm) Process(payment *storage.Payment) {
	payment.TxId = p.TxId
	payment.PaidAt = time.Now()
	payment.PaidBy = int(storage.PaidByPaymentSettings)
	payment.Status = storage.PaymentStatusPaid
	payment.ConvertRate = p.ConvertRate
	payment.ConvertTime = p.ConvertTime
	payment.ExpectedAmount = p.ExpectedAmount
	payment.PaymentMethod = p.PaymentMethod
	payment.PaymentAddress = p.PaymentAddress
}

type PaymentRequestRate struct {
	Id             uint64       `json:"id" validate:"required"`
	Token          string       `json:"token"`
	PaymentMethod  utils.Method `json:"paymentMethod"`
	PaymentAddress string       `json:"paymentAddress"`
	Exchange       string       `json:"exchange"`
}

type ListPaymentSettingRequest struct {
	List []ApproversSettingRequest `json:"list"`
}

type ApproversSettingRequest struct {
	ApproverIds []uint64 `json:"approverIds"`
	SendUserId  uint64   `json:"sendUserId"`
	ShowCost    bool     `json:"showCost"`
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
type PaymentSummary struct {
	RequestReceived uint64  `json:"requestReceived"`
	RequestSent     uint64  `json:"requestSent"`
	RequestPaid     uint64  `json:"requestPaid"`
	TotalPaid       float64 `json:"totalPaid"`
	TotalReceived   float64 `json:"totalReceived"`
}

type SummaryFilter struct {
	Month uint64 `json:"month"`
	Ids   string `json:"ids"`
}

type PaymentReport struct {
	Month        string              `json:"month"`
	PaymentUnits []PaymentReportUnit `json:"paymentUnits"`
}

type PaymentReportUnit struct {
	DisplayName    string       `json:"displayName"`
	Amount         float64      `json:"amount"`
	ExpectedAmount float64      `json:"expectedAmount"`
	PaymentMethod  utils.Method `json:"paymentMethod"`
}

type InvoiceReport struct {
	Project      string              `json:"project"`
	DisplayName  string              `json:"displayName"`
	InvoiceUnits []InvoiceReportUnit `json:"invoiceUnits"`
}

type InvoiceReportUnit struct {
	Date        string  `json:"date"`
	Hours       float64 `json:"hours"`
	Description string  `json:"description"`
}
type AddressReport struct {
	PaymentMethod string              `json:"paymentMethod"`
	DisplayName   string              `json:"displayName"`
	AddressUnits  []AddressReportUnit `json:"addressUnits"`
}

type AddressReportUnit struct {
	DateTime       string  `json:"dateTime"`
	Amount         float64 `json:"amount"`
	ExpectedAmount float64 `json:"expectedAmount"`
}

type ReportFilter struct {
	StartDate  time.Time
	EndDate    time.Time
	MemberIds  string
	ProjectIds string
}

type ProductForDelete struct {
	DeleteAll bool   `json:"deleteAll"`
	Id        uint64 `json:"id"`
	OrderId   uint64 `json:"orderId"`
	ProductId uint64 `json:"productId"`
}
