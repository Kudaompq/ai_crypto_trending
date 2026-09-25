package service

var binanceUSDMIntervals = map[string]struct{}{
	"1m": {}, "3m": {}, "5m": {}, "15m": {}, "30m": {},
	"1h": {}, "2h": {}, "4h": {}, "6h": {}, "8h": {}, "12h": {},
	"1d": {}, "3d": {}, "1w": {}, "1M": {},
}

// IsSupportedKlineInterval reports whether interval is native to Binance USDⓈ-M.
func IsSupportedKlineInterval(interval string) bool {
	_, ok := binanceUSDMIntervals[interval]
	return ok
}
