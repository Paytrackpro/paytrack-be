package service

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Paytrackpro/paytrack-be/utils"
)

const (
	binancePriceURL = "https://api.binance.com/api/v3/ticker/price"
	kucoinPriceURL  = "https://api.kucoin.com/api/v1/market/stats"
	coinMaketCapURL = "https://pro-api.coinmarketcap.com/v2/tools/price-conversion"
	bittrexURL      = "https://api.bittrex.com/v3/markets/"
	mexcPriceURL    = "https://api.mexc.com/api/v3/ticker/price"
)

const (
	Bittrex       = "bittrex"
	Binance       = "binance"
	Kucoin        = "kucoin"
	Mexc          = "mexc"
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

type KucoinPriceResponse struct {
	Code string             `json:"code"`
	Data KucoinResponseData `json:"data"`
}

type KucoinResponseData struct {
	Symbol           string `json:"symbol"`
	Time             int64  `json:"time"`
	Buy              string `json:"buy"`
	Sell             string `json:"sell"`
	ChangeRate       string `json:"changeRate"`
	ChangePrice      string `json:"changePrice"`
	High             string `json:"high"`
	Low              string `json:"low"`
	Vol              string `json:"vol"`
	VolValue         string `json:"volValue"`
	Last             string `json:"last"`
	AveragePrice     string `json:"averagePrice"`
	TakerFeeRate     string `json:"takerFeeRate"`
	MakerFeeRate     string `json:"makerFeeRate"`
	TakerCoefficient string `json:"takerCoefficient"`
	MakerCoefficient string `json:"makerCoefficient"`
}

func (s *Service) GetRate(currency utils.Method) (float64, error) {
	switch s.exchange {
	case Binance:
		return s.getBinancePrice(currency)
	case Kucoin:
		return s.getKucoinPrice(currency)
	case Coinmarketcap:
		return s.getCoinMarketCapPrice(currency)
	case Bittrex:
		return s.getBittrexPrice(currency)
	case Mexc:
		return s.getMexcPrice(currency)
	default:
		return 0, fmt.Errorf("exchange not set")
	}
}

func (s *Service) GetBTCBulkRate() (float64, error) {
	if utils.IsEmpty(s.ExchangeList) {
		return 0, fmt.Errorf("%s", "Get BTC rate failed")
	}
	exchangeLists := strings.Split(s.ExchangeList, ",")
	for _, exchange := range exchangeLists {
		exchange = strings.TrimSpace(exchange)
		rate, err := s.GetExchangeRate(exchange, utils.PaymentTypeBTC)
		if err == nil {
			return rate, nil
		}
	}
	return 0, fmt.Errorf("%s", "Get BTC rate failed")
}

func (s *Service) GetExchangeRate(exchange string, currency utils.Method) (float64, error) {
	switch exchange {
	case Binance:
		return s.getBinancePrice(currency)
	case Kucoin:
		return s.getKucoinPrice(currency)
	case Coinmarketcap:
		return s.getCoinMarketCapPrice(currency)
	case Bittrex:
		return s.getBittrexPrice(currency)
	case Mexc:
		return s.getMexcPrice(currency)
	default:
		return 0, fmt.Errorf("exchange not set")
	}
}

// GetPrice get the price of the cryptocurrency based on kucoin api
// at the moment, the use of kucoin is simple, so we build a simple function for it
func (s *Service) getKucoinPrice(currency utils.Method) (float64, error) {
	var symbol = fmt.Sprintf("%s-USDT", strings.ToUpper(currency.String()))
	query := map[string]string{
		"symbol": symbol,
	}
	req := &ReqConfig{
		Method:  http.MethodGet,
		HttpUrl: kucoinPriceURL,
		Payload: query,
	}
	var kuCoinRes KucoinPriceResponse
	if err := HttpRequest(req, &kuCoinRes); err != nil {
		return 0, err
	}

	if kuCoinRes.Code != "200000" {
		return 0, fmt.Errorf("Get Kucoin %s price failed", currency)
	}
	lastPrice, parseErr := strconv.ParseFloat(kuCoinRes.Data.Last, 64)
	if parseErr != nil {
		return 0, parseErr
	}
	return lastPrice, nil
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

// GetPrice get the price of the cryptocurrency based on binance api
// at the moment, the use of binance is simple, so we build a simple function for it
func (s *Service) getMexcPrice(currency utils.Method) (float64, error) {
	var symbol = fmt.Sprintf("%sUSDT", strings.ToUpper(currency.String()))
	query := map[string]string{
		"symbol": symbol,
	}
	req := &ReqConfig{
		Method:  http.MethodGet,
		HttpUrl: mexcPriceURL,
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

func (s *Service) IsValidExchange(exchange string) bool {
	//Check valid exchange with DCR rate
	_, err := s.GetExchangeRate(exchange, utils.PaymentTypeDCR)
	return err == nil
}

type Map map[string]interface{}

func (s *Service) NotifyCryptoPriceChanged() {
	for range time.Tick(time.Second * 7) {
		for _, currency := range []utils.Method{utils.PaymentTypeBTC, utils.PaymentTypeDCR, utils.PaymentTypeLTC} {
			if utils.IsEmpty(s.ExchangeList) {
				continue
			}
			exchangeLists := strings.Split(s.ExchangeList, ",")
			dataMap := make(map[string]Map)
			for _, exchange := range exchangeLists {
				if utils.IsEmpty(exchange) {
					continue
				}
				rate, err := s.GetExchangeRate(exchange, currency)
				if err != nil {
					continue
				}
				exchangeData := Map{
					"rate":        rate,
					"convertTime": time.Now(),
				}
				dataMap[exchange] = exchangeData
			}

			s.socket.BroadcastToRoom("", "exchangeRate", currency.String(), dataMap)
		}
	}
}
