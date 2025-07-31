package utils

// CoinNetworkSupport represents the supported networks for a specific coin
type CoinNetworkSupport struct {
	Coin     string        `json:"coin"`
	Networks []NetworkInfo `json:"networks"`
}

// GetSupportedCoinsAndNetworks returns all supported coin and network combinations
func GetSupportedCoinsAndNetworks() []CoinNetworkSupport {
	return []CoinNetworkSupport{
		{
			Coin: "BTC",
			Networks: []NetworkInfo{
				NetworkBTC.Info(),
				NetworkBEP20.Info(),
				NetworkERC20.Info(),
			},
		},
		{
			Coin: "LTC",
			Networks: []NetworkInfo{
				NetworkLTC.Info(),
				NetworkBEP20.Info(),
			},
		},
		{
			Coin: "DCR",
			Networks: []NetworkInfo{
				NetworkDCR.Info(),
			},
		},
		{
			Coin: "ETH",
			Networks: []NetworkInfo{
				NetworkBEP20.Info(),
				NetworkERC20.Info(),
			},
		},
		{
			Coin: "USDT",
			Networks: []NetworkInfo{
				NetworkBEP20.Info(),
				NetworkSolana.Info(),
				NetworkERC20.Info(),
			},
		},
	}
}

// GetSupportedNetworksForCoin returns supported networks for a specific coin
func GetSupportedNetworksForCoin(coin string) ([]NetworkInfo, bool) {
	supportedCoins := GetSupportedCoinsAndNetworks()
	for _, coinSupport := range supportedCoins {
		if coinSupport.Coin == coin {
			return coinSupport.Networks, true
		}
	}
	return nil, false
}

// IsCoinNetworkSupported checks if a coin-network combination is supported
func IsCoinNetworkSupported(coin, networkCode string) bool {
	networks, exists := GetSupportedNetworksForCoin(coin)
	if !exists {
		return false
	}
	
	for _, network := range networks {
		if network.Code == networkCode {
			return true
		}
	}
	return false
}

// GetAllSupportedCoins returns a list of all supported coins
func GetAllSupportedCoins() []string {
	supportedCoins := GetSupportedCoinsAndNetworks()
	coins := make([]string, len(supportedCoins))
	for i, coinSupport := range supportedCoins {
		coins[i] = coinSupport.Coin
	}
	return coins
}

// GetAllSupportedMethods returns a list of all supported Method types
func GetAllSupportedMethods() []Method {
	return []Method{
		PaymentTypeBTC,
		PaymentTypeLTC,
		PaymentTypeDCR,
		PaymentTypeETH,
		PaymentTypeUSDT,
	}
}

// IsMethodSupported checks if a Method type is supported
func IsMethodSupported(method Method) bool {
	supportedMethods := GetAllSupportedMethods()
	for _, supportedMethod := range supportedMethods {
		if method == supportedMethod {
			return true
		}
	}
	return false
}

// MethodFromCoin converts a coin string to Method type
func MethodFromCoin(coin string) Method {
	switch coin {
	case "BTC":
		return PaymentTypeBTC
	case "LTC":
		return PaymentTypeLTC
	case "DCR":
		return PaymentTypeDCR
	case "ETH":
		return PaymentTypeETH
	case "USDT":
		return PaymentTypeUSDT
	default:
		return PaymentTypeNotSet
	}
}