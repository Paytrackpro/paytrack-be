package utils

import (
	"encoding/json"
)

type Method int
type Type int

const (
	PaymentTypeNotSet Method = iota
	PaymentTypeBTC
	PaymentTypeLTC
	PaymentTypeDCR
)

const (
	PaymentSystem Type = iota
	PaymentUrl
)

func (m Method) String() string {
	switch m {
	case PaymentTypeBTC:
		return "btc"
	case PaymentTypeLTC:
		return "ltc"
	case PaymentTypeDCR:
		return "dcr"
	}
	return "none"
}

func (m Method) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.String())
}

func (m *Method) UnmarshalText(val []byte) error {
	switch string(val) {
	case "btc":
		*m = PaymentTypeBTC
		return nil
	case "ltc":
		*m = PaymentTypeLTC
		return nil
	case "dcr":
		*m = PaymentTypeDCR
		return nil
	}
	*m = PaymentTypeNotSet
	return nil
}

func (m *Method) UnmarshalJSON(v []byte) error {
	var val string
	if err := json.Unmarshal(v, &val); err != nil {
		return err
	}
	return m.UnmarshalText([]byte(val))
}
