package utils

import (
	"encoding/json"
)

type CoinInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Method string
type Type int

const (
	PaymentTypeNotSet Method = ""
	PaymentTypeBTC    Method = "btc"
	PaymentTypeLTC    Method = "ltc"
	PaymentTypeDCR    Method = "dcr"
	PaymentTypeETH    Method = "eth"
	PaymentTypeUSDT   Method = "usdt"
)

const (
	PaymentSystem Type = iota
	PaymentUrl
)

// Info returns the coin information with both code and display name
func (m Method) Info() CoinInfo {
	switch m {
	case PaymentTypeBTC:
		return CoinInfo{Code: "btc", Name: "Bitcoin"}
	case PaymentTypeLTC:
		return CoinInfo{Code: "ltc", Name: "Litecoin"}
	case PaymentTypeDCR:
		return CoinInfo{Code: "dcr", Name: "Decred"}
	case PaymentTypeETH:
		return CoinInfo{Code: "eth", Name: "Ethereum"}
	case PaymentTypeUSDT:
		return CoinInfo{Code: "usdt", Name: "Tether USD"}
	default:
		return CoinInfo{Code: string(m), Name: string(m)}
	}
}

func (m Method) String() string {
	if m == PaymentTypeNotSet {
		return "none"
	}
	return string(m)
}

// MethodFromCode returns the Method type from a code string
func MethodFromCode(code string) Method {
	return Method(code)
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
	case "eth":
		*m = PaymentTypeETH
		return nil
	case "usdt":
		*m = PaymentTypeUSDT
		return nil
	case "none", "":
		*m = PaymentTypeNotSet
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
