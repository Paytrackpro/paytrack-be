package utils

import (
	"fmt"
	"strconv"
)

func BtcRoundFloat(f float64) float64 {
	fStr := fmt.Sprintf("%.8f", f)
	f, _ = strconv.ParseFloat(fStr, 64)
	return f
}

func Uint64(str string) uint64 {
	num, _ := strconv.Atoi(str)
	return uint64(num)
}
