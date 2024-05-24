package service

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.cryptopower.dev/mgmt-ng/be/utils"
)

const (
	binancePriceURL = "https://api.binance.com/api/v3/ticker/price"
	coinMaketCapURL = "https://pro-api.coinmarketcap.com/v2/tools/price-conversion"
	bittrexURL      = "https://api.bittrex.com/v3/markets/"
)

const (
	Bittrex       = "bittrex"
	Binance       = "binance"
	Coinmarketcap = "coinmarketcap"
)

type ticker struct {
	Symbol string  `json:"symbol"`
	Price  float64 `json:"price,string"`
}

type BittrexPrice struct {
	Price   float64 `json:"lastTradeRate,string"`
	BidRate float64 `json:"bidRate,string"`
	AskRate float64 `json:"askRate,string"`
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

func (s *Service) GetRate(currency utils.Method) (float64, error) {
	switch s.exchange {
	case Binance:
		return s.getBinancePrice(currency)
	case Coinmarketcap:
		return s.getCoinMarketCapPrice(currency)
	case Bittrex:
		return s.getBittrexPrice(currency)
	default:
		return 0, fmt.Errorf("exchange not set")
	}
}

// GetPrice get the price of the cryptocurrency based on binance api
// at the moment, the use of binance is simple, so we build a simple function for it
func (s *Service) getBinancePrice(currency utils.Method) (float64, error) {
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

func (s *Service) getCoinMarketCapPrice(currency utils.Method) (float64, error) {
	query := map[string]string{
		"symbol":  strings.ToUpper(currency.String()),
		"convert": "USD",
		"amount":  "1",
	}

	header := map[string]string{
		"X-CMC_PRO_API_KEY": s.coinMaketCapKey,
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

func (s *Service) getBittrexPrice(currency utils.Method) (float64, error) {
	req := &ReqConfig{
		Method:  http.MethodGet,
		HttpUrl: fmt.Sprintf("%s%s-USDT%s", bittrexURL, strings.ToUpper(currency.String()), "/ticker"),
	}

	var res BittrexPrice
	if err := HttpRequest(req, &res); err != nil {
		return 0, err
	}

	return res.Price, nil
}

type Map map[string]interface{}

func (s *Service) NotifyCryptoPriceChanged() {
	for range time.Tick(time.Second * 5) {
		for _, currency := range []utils.Method{utils.PaymentTypeBTC, utils.PaymentTypeDCR, utils.PaymentTypeLTC} {
			rate, err := s.GetRate(currency)
			if err != nil {
				fmt.Printf("error getting %s rate: %v\n", currency.String(), err)
				continue
			}
			s.socket.BroadcastToRoom("", "exchangeRate", currency.String(), Map{
				"currency":    currency.String(),
				"rate":        rate,
				"convertTime": time.Now(),
			})
		}
	}
}
