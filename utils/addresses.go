package utils

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	btcChaincfg "github.com/btcsuite/btcd/chaincfg"
	"github.com/decred/dcrd/dcrutil"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gagliardetto/solana-go"
	ltcChaincfg "github.com/ltcsuite/ltcd/chaincfg"
	"github.com/ltcsuite/ltcd/ltcutil"
)

type NetworkInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type Network string

const (
	NetworkBTC    Network = "btc"
	NetworkLTC    Network = "ltc"
	NetworkDCR    Network = "dcr"
	NetworkSolana Network = "solana"
	NetworkBEP20  Network = "bep20"
	NetworkERC20  Network = "erc20"
)

// Info returns the network information with both code and display name
func (n Network) Info() NetworkInfo {
	switch n {
	case NetworkBTC:
		return NetworkInfo{Code: "btc", Name: "Bitcoin"}
	case NetworkLTC:
		return NetworkInfo{Code: "ltc", Name: "Litecoin"}
	case NetworkDCR:
		return NetworkInfo{Code: "dcr", Name: "Decred"}
	case NetworkSolana:
		return NetworkInfo{Code: "solana", Name: "Solana"}
	case NetworkBEP20:
		return NetworkInfo{Code: "bep20", Name: "BNB Smart Chain (BEP20)"}
	case NetworkERC20:
		return NetworkInfo{Code: "erc20", Name: "Ethereum (ERC20)"}
	default:
		return NetworkInfo{Code: string(n), Name: string(n)}
	}
}

// NetworkFromCode returns the Network type from a code string
func NetworkFromCode(code string) Network {
	return Network(code)
}

func VerifyAddress(address string, network Network) error {
	switch network {
	case NetworkBTC:
		btcMainNetParams := &btcChaincfg.MainNetParams
		addr, err := btcutil.DecodeAddress(address, btcMainNetParams)
		if err != nil {
			fmt.Println("BTC Address invalid with error: ", err)
			return err
		}

		if !addr.IsForNet(btcMainNetParams) {
			fmt.Println("Address valid but not for MainNet.")
			return fmt.Errorf("address '%s' valid but not for MainNet", address)
		}
		return nil
	case NetworkLTC:
		ltcMainNetParams := &ltcChaincfg.MainNetParams
		addr, err := ltcutil.DecodeAddress(address, ltcMainNetParams)
		if err != nil {
			fmt.Println("LTC Address invalid with error: ", err)
			return err
		}

		if !addr.IsForNet(ltcMainNetParams) {
			fmt.Println("Address valid but not for MainNet")
			return fmt.Errorf("address '%s' valid but not for MainNet", address)
		}
		return nil
	case NetworkDCR:
		_, err := dcrutil.DecodeAddress(address)
		if err != nil {
			fmt.Println("DCR Address invalid with error: ", err)
			return err
		}

		return nil
	case NetworkSolana:
		_, err := solana.PublicKeyFromBase58(address)
		if err != nil {
			fmt.Println("Solana Address invalid with error: ", err)
			return err
		}
		return nil
	case NetworkBEP20:
		isValid := common.IsHexAddress(address)
		if !isValid {
			fmt.Println("BEP20 Address invalid: ")
			return fmt.Errorf("address '%s' valid but not for MainNet", address)
		}
		return nil
	case NetworkERC20:
		isValid := common.IsHexAddress(address)
		if !isValid {
			fmt.Println("ERC20 Address invalid: ")
			return fmt.Errorf("address '%s' valid but not for MainNet", address)
		}
		return nil
	}
	return nil
}
