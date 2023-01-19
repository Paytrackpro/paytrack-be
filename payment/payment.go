package payment

type Type int

const (
	PaymentTypeNotSet Type = iota
	PaymentTypeBTC
	PaymentTypeLTC
	PaymentTypeDCR
)
