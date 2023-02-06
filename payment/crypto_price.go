package payment

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
func GetPrice(currency Method) (float64, error) {
	var symbol = fmt.Sprintf("%sUSDT", strings.ToUpper(currency.String()))
	res, err := http.Get(fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", symbol))
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	var t ticker
	err = parseBinanceResponse(body, &t)
	return t.Price, err
}

func parseBinanceResponse(r []byte, obj interface{}) error {
	var bErr binanceError
	if err := json.Unmarshal(r, &bErr); err == nil && bErr.Code != 0 {
		return fmt.Errorf(bErr.Message)
	}
	return json.Unmarshal(r, &obj)
}
