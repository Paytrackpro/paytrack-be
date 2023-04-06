package service

import (
	"fmt"
	"net/http"
	"strings"

	"code.cryptopower.dev/mgmt-ng/be/utils"
)

const (
	binancePriceURL = "https://api.binance.com/api/v3/ticker/price"
)

type ticker struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

type binanceError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

// GetPrice get the price of the cryptocurrency based on binance api
// at the moment, the use of binance is simple, so we build a simple function for it
func GetPrice(currency utils.Method) (float64, error) {
	var symbol = fmt.Sprintf("%sUSDT", strings.ToUpper(currency.String()))
	query := map[string]string{
		"symbol": symbol,
	}
	req := &ReqConfig{
		Method:  http.MethodGet,
		HttpUrl: binancePriceURL,
		Payload: query,
	}
	var t ticker
	if err := HttpRequest(req, &t); err != nil {
		return 0, err
	}

	return t.Price, nil
}
