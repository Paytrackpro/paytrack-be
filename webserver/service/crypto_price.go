package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"code.cryptopower.dev/mgmt-ng/be/utils"
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
	fmt.Println("------Price---->", t.Price)
	fmt.Println("------Symbol---->", t.Symbol)
	return t.Price, err
}

func parseBinanceResponse(r []byte, obj interface{}) error {
	var bErr binanceError
	if err := json.Unmarshal(r, &bErr); err == nil && bErr.Code != 0 {
		return fmt.Errorf(bErr.Message)
	}
	fmt.Println("------GetPrice---->", string(r))
	return json.Unmarshal(r, &obj)
}

func GetPrice1(currency utils.Method) (float64, error) {
	var symbol = fmt.Sprintf("%sUSDT", strings.ToUpper(currency.String()))
	query := map[string]string{
		"symbol": symbol,
	}
	req := &ReqConfig{
		Method:  http.MethodGet,
		HttpUrl: "https://api.binance.com/api/v3/ticker/price",
		Payload: query,
	}
	var t ticker
	if err := HttpRequest(req, &t); err != nil {
		return 0, err
	}
	fmt.Println("------Price---->", t.Price)
	fmt.Println("------Symbol---->", t.Symbol)
	return t.Price, nil
}
