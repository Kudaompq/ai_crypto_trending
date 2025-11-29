package model

// Candle represents a single K-line/candlestick
type Candle struct {
	Timestamp int64   `json:"timestamp"` // Unix timestamp in milliseconds
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
}

// KlineData represents the complete K-line dataset
type KlineData struct {
	Symbol   string    `json:"symbol"`
	Interval string    `json:"interval"`
	Data     []Candle  `json:"data"`
}
