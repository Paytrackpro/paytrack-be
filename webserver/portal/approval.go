package portal

type ApprovalRequest struct {
	PaymentId uint64 `json:"paymentId"`
	Status    uint64 `json:"status"`
}
