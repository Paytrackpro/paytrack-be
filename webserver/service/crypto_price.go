package service

import (
	"fmt"
	"net/http"
	"strings"

	"code.cryptopower.dev/mgmt-ng/be/utils"
)

const (
	binancePriceURL = "https://api.binance.com/api/v3/ticker/price"
	coinMaketCapURL = "https://pro-api.coinmarketcap.com/v2/tools/price-conversion"
)

const (
	coinMaketCapKey = "652e1706-18a5-40dc-9a60-2df20cd6a7f9"
)

type ticker struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

type binanceError struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

type CoinMarketCapData struct {
	Status struct {
		ErrorCode int    `json:"error_code"`
		ErrorMess string `json:"error_message"`
	} `json:"status"`
	Data []CoinMarketCapConvert `json:"data"`
}

type CoinMarketCapConvert struct {
	Amount int `json:"amount"`
	Quote  struct {
		USD struct {
			Price float64 `json:"price"`
		} `json:"USD"`
	} `json:"quote"`
}

// GetPrice get the price of the cryptocurrency based on binance api
// at the moment, the use of binance is simple, so we build a simple function for it
func GetRate(currency utils.Method) (float64, error) {
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

func GetCoinMarketCapRate(currency utils.Method) (float64, error) {
	query := map[string]string{
		"symbol":  strings.ToUpper(currency.String()),
		"convert": "USD",
		"amount":  "1",
	}

	header := map[string]string{
		"X-CMC_PRO_API_KEY": coinMaketCapKey,
	}

	req := &ReqConfig{
		Method:  http.MethodGet,
		HttpUrl: coinMaketCapURL,
		Payload: query,
		Header:  header,
	}

	var res CoinMarketCapData
	if err := HttpRequest(req, &res); err != nil {
		return 0, err
	}

	return res.Data[0].Quote.USD.Price, nil
}
