package model

type OpenInterestSample struct {
	Timestamp int64   `json:"timestamp"`
	Quantity  float64 `json:"quantity"`
	Value     float64 `json:"value"`
}

type OpenInterestData struct {
	Symbol   string               `json:"symbol"`
	Interval string               `json:"interval"`
	Data     []OpenInterestSample `json:"data"`
}
