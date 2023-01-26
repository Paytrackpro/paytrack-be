package payment

type Method int

const (
	PaymentTypeNotSet Method = iota
	PaymentTypeBTC
	PaymentTypeLTC
	PaymentTypeDCR
)
