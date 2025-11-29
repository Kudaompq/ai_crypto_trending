package indicator

import (
	"math"

	"github.com/kudaompq/ai_trending/backend/internal/model"
)

// IdentifyPatterns identifies candlestick patterns in the given candles
func IdentifyPatterns(candles []model.Candle, trend string) []model.CandlestickPattern {
	patterns := make([]model.CandlestickPattern, 0)
	n := len(candles)

	if n < 1 {
		return patterns
	}

	curr := candles[n-1]

	// Single candle patterns
	if IsHammer(curr, trend) {
		patterns = append(patterns, model.CandlestickPattern{
			Pattern:     "锤子线",
			Type:        "反转",
			Direction:   "看涨",
			Position:    0,
			Reliability: 0.80,
			Description: "下降趋势底部出现长下影线，买方力量增强",
		})
	}

	if IsHangingMan(curr, trend) {
		patterns = append(patterns, model.CandlestickPattern{
			Pattern:     "上吊线",
			Type:        "反转",
			Direction:   "看跌",
			Position:    0,
			Reliability: 0.70,
			Description: "上升趋势顶部出现长下影线，需次日确认",
		})
	}

	if IsInvertedHammer(curr, trend) {
		patterns = append(patterns, model.CandlestickPattern{
			Pattern:     "倒锤子线",
			Type:        "反转",
			Direction:   "看涨",
			Position:    0,
			Reliability: 0.65,
			Description: "下降趋势底部出现长上影线，需次日阳线确认",
		})
	}

	if IsShootingStar(curr, trend) {
		patterns = append(patterns, model.CandlestickPattern{
			Pattern:     "射击之星",
			Type:        "反转",
			Direction:   "看跌",
			Position:    0,
			Reliability: 0.75,
			Description: "上升趋势顶部出现长上影线，价格冲高回落",
		})
	}

	isDoji, dojiType := IsDoji(curr)
	if isDoji {
		reliability := 0.70
		description := "市场犹豫不决，趋势可能改变"
		
		if dojiType == "Dragonfly" && trend == "下降" {
			reliability = 0.75
			description = "蜻蜓十字星出现在底部，看涨信号"
		} else if dojiType == "Gravestone" && trend == "上升" {
			reliability = 0.75
			description = "墓碑十字星出现在顶部，看跌信号"
		}

		patterns = append(patterns, model.CandlestickPattern{
			Pattern:     "十字星 (" + dojiType + ")",
			Type:        "反转",
			Direction:   "中性",
			Position:    0,
			Reliability: reliability,
			Description: description,
		})
	}

	// Two candle patterns
	if n >= 2 {
		prev := candles[n-2]

		if IsBullishEngulfing(prev, curr, trend) {
			patterns = append(patterns, model.CandlestickPattern{
				Pattern:     "看涨吞没",
				Type:        "反转",
				Direction:   "看涨",
				Position:    -1,
				Reliability: 0.90,
				Description: "大阳线吞没前阴线，强烈看涨信号",
			})
		}

		if IsBearishEngulfing(prev, curr, trend) {
			patterns = append(patterns, model.CandlestickPattern{
				Pattern:     "看跌吞没",
				Type:        "反转",
				Direction:   "看跌",
				Position:    -1,
				Reliability: 0.90,
				Description: "大阴线吞没前阳线，强烈看跌信号",
			})
		}

		if IsPiercingPattern(prev, curr, trend) {
			patterns = append(patterns, model.CandlestickPattern{
				Pattern:     "刺透形态",
				Type:        "反转",
				Direction:   "看涨",
				Position:    -1,
				Reliability: 0.80,
				Description: "跳空低开后强力反弹，买方占据优势",
			})
		}

		if IsDarkCloudCover(prev, curr, trend) {
			patterns = append(patterns, model.CandlestickPattern{
				Pattern:     "乌云盖顶",
				Type:        "反转",
				Direction:   "看跌",
				Position:    -1,
				Reliability: 0.80,
				Description: "跳空高开后回落，卖方开始占据优势",
			})
		}

		isHarami, haramiType := IsHarami(prev, curr)
		if isHarami {
			direction := "中性"
			if haramiType == "Bullish" {
				direction = "看涨"
			} else if haramiType == "Bearish" {
				direction = "看跌"
			}

			patterns = append(patterns, model.CandlestickPattern{
				Pattern:     "孕线",
				Type:        "反转",
				Direction:   direction,
				Position:    -1,
				Reliability: 0.70,
				Description: "小K线包含在大K线内，趋势可能反转",
			})
		}
	}

	// Three candle patterns
	if n >= 3 {
		c1 := candles[n-3]
		c2 := candles[n-2]
		c3 := candles[n-1]

		if IsMorningStar(c1, c2, c3, trend) {
			patterns = append(patterns, model.CandlestickPattern{
				Pattern:     "启明星",
				Type:        "反转",
				Direction:   "看涨",
				Position:    -2,
				Reliability: 0.95,
				Description: "底部反转的最强信号之一",
			})
		}

		if IsEveningStar(c1, c2, c3, trend) {
			patterns = append(patterns, model.CandlestickPattern{
				Pattern:     "黄昏星",
				Type:        "反转",
				Direction:   "看跌",
				Position:    -2,
				Reliability: 0.95,
				Description: "顶部反转的最强信号之一",
			})
		}

		if IsThreeWhiteSoldiers(c1, c2, c3, trend) {
			patterns = append(patterns, model.CandlestickPattern{
				Pattern:     "三个白兵",
				Type:        "反转",
				Direction:   "看涨",
				Position:    -2,
				Reliability: 0.85,
				Description: "连续三根阳线，买方持续施压",
			})
		}

		if IsThreeBlackCrows(c1, c2, c3, trend) {
			patterns = append(patterns, model.CandlestickPattern{
				Pattern:     "三只乌鸦",
				Type:        "反转",
				Direction:   "看跌",
				Position:    -2,
				Reliability: 0.85,
				Description: "连续三根阴线，卖方持续施压",
			})
		}
	}

	return patterns
}

// IsHammer checks if the candle is a hammer pattern
func IsHammer(candle model.Candle, trend string) bool {
	bodyLength := math.Abs(candle.Close - candle.Open)
	lowerShadow := math.Min(candle.Open, candle.Close) - candle.Low
	upperShadow := candle.High - math.Max(candle.Open, candle.Close)
	totalLength := candle.High - candle.Low

	if totalLength == 0 {
		return false
	}

	return trend == "下降" &&
		lowerShadow >= bodyLength*2 &&
		upperShadow <= bodyLength*0.1 &&
		bodyLength/totalLength < 0.3
}

// IsHangingMan checks if the candle is a hanging man pattern
func IsHangingMan(candle model.Candle, trend string) bool {
	bodyLength := math.Abs(candle.Close - candle.Open)
	lowerShadow := math.Min(candle.Open, candle.Close) - candle.Low
	upperShadow := candle.High - math.Max(candle.Open, candle.Close)

	return trend == "上升" &&
		lowerShadow >= bodyLength*2 &&
		upperShadow <= bodyLength*0.1
}

// IsInvertedHammer checks if the candle is an inverted hammer pattern
func IsInvertedHammer(candle model.Candle, trend string) bool {
	bodyLength := math.Abs(candle.Close - candle.Open)
	upperShadow := candle.High - math.Max(candle.Open, candle.Close)
	lowerShadow := math.Min(candle.Open, candle.Close) - candle.Low

	return trend == "下降" &&
		upperShadow >= bodyLength*2 &&
		lowerShadow <= bodyLength*0.1
}

// IsShootingStar checks if the candle is a shooting star pattern
func IsShootingStar(candle model.Candle, trend string) bool {
	bodyLength := math.Abs(candle.Close - candle.Open)
	upperShadow := candle.High - math.Max(candle.Open, candle.Close)
	lowerShadow := math.Min(candle.Open, candle.Close) - candle.Low

	return trend == "上升" &&
		upperShadow >= bodyLength*2 &&
		lowerShadow <= bodyLength*0.1
}

// IsDoji checks if the candle is a doji pattern
func IsDoji(candle model.Candle) (bool, string) {
	bodyLength := math.Abs(candle.Close - candle.Open)
	totalLength := candle.High - candle.Low
	upperShadow := candle.High - math.Max(candle.Open, candle.Close)
	lowerShadow := math.Min(candle.Open, candle.Close) - candle.Low

	if totalLength == 0 || bodyLength/totalLength > 0.05 {
		return false, ""
	}

	// Dragonfly Doji
	if lowerShadow > totalLength*0.6 && upperShadow < totalLength*0.1 {
		return true, "Dragonfly"
	}

	// Gravestone Doji
	if upperShadow > totalLength*0.6 && lowerShadow < totalLength*0.1 {
		return true, "Gravestone"
	}

	// Long-Legged Doji
	if lowerShadow > totalLength*0.3 && upperShadow > totalLength*0.3 {
		return true, "Long-Legged"
	}

	return true, "Standard"
}

// IsBullishEngulfing checks for bullish engulfing pattern
func IsBullishEngulfing(prev, curr model.Candle, trend string) bool {
	prevBody := math.Abs(prev.Close - prev.Open)
	currBody := math.Abs(curr.Close - curr.Open)

	return trend == "下降" &&
		prev.Close < prev.Open && // Previous is bearish
		curr.Close > curr.Open && // Current is bullish
		curr.Open < prev.Close && // Opens below previous close
		curr.Close > prev.Open && // Closes above previous open
		currBody > prevBody*1.5 // Significantly larger body
}

// IsBearishEngulfing checks for bearish engulfing pattern
func IsBearishEngulfing(prev, curr model.Candle, trend string) bool {
	prevBody := math.Abs(prev.Close - prev.Open)
	currBody := math.Abs(curr.Close - curr.Open)

	return trend == "上升" &&
		prev.Close > prev.Open && // Previous is bullish
		curr.Close < curr.Open && // Current is bearish
		curr.Open > prev.Close && // Opens above previous close
		curr.Close < prev.Open && // Closes below previous open
		currBody > prevBody*1.5 // Significantly larger body
}

// IsPiercingPattern checks for piercing pattern
func IsPiercingPattern(prev, curr model.Candle, trend string) bool {
	if prev.Open == prev.Close {
		return false
	}

	prevBody := prev.Open - prev.Close
	penetration := (curr.Close - prev.Close) / prevBody

	return trend == "下降" &&
		prev.Close < prev.Open && // Previous is bearish
		curr.Close > curr.Open && // Current is bullish
		curr.Open < prev.Low && // Gaps down
		curr.Close > prev.Close && // Closes above previous close
		curr.Close < prev.Open && // But below previous open
		penetration >= 0.5 // Penetrates at least 50%
}

// IsDarkCloudCover checks for dark cloud cover pattern
func IsDarkCloudCover(prev, curr model.Candle, trend string) bool {
	if prev.Close == prev.Open {
		return false
	}

	prevBody := prev.Close - prev.Open
	penetration := (prev.Close - curr.Close) / prevBody

	return trend == "上升" &&
		prev.Close > prev.Open && // Previous is bullish
		curr.Close < curr.Open && // Current is bearish
		curr.Open > prev.High && // Gaps up
		curr.Close < prev.Close && // Closes below previous close
		curr.Close > prev.Open && // But above previous open
		penetration >= 0.5 // Penetrates at least 50%
}

// IsHarami checks for harami pattern
func IsHarami(prev, curr model.Candle) (bool, string) {
	prevMax := math.Max(prev.Open, prev.Close)
	prevMin := math.Min(prev.Open, prev.Close)
	currMax := math.Max(curr.Open, curr.Close)
	currMin := math.Min(curr.Open, curr.Close)

	if currMax < prevMax && currMin > prevMin {
		if prev.Close < prev.Open && curr.Close > curr.Open {
			return true, "Bullish"
		} else if prev.Close > prev.Open && curr.Close < curr.Open {
			return true, "Bearish"
		}
	}
	return false, ""
}

// IsMorningStar checks for morning star pattern
func IsMorningStar(c1, c2, c3 model.Candle, trend string) bool {
	body1 := math.Abs(c1.Close - c1.Open)
	body2 := math.Abs(c2.Close - c2.Open)

	return trend == "下降" &&
		c1.Close < c1.Open && // First is bearish
		body2 < body1*0.3 && // Second is small
		c3.Close > c3.Open && // Third is bullish
		c3.Close > (c1.Open+c1.Close)/2 // Penetrates first candle
}

// IsEveningStar checks for evening star pattern
func IsEveningStar(c1, c2, c3 model.Candle, trend string) bool {
	body1 := math.Abs(c1.Close - c1.Open)
	body2 := math.Abs(c2.Close - c2.Open)

	return trend == "上升" &&
		c1.Close > c1.Open && // First is bullish
		body2 < body1*0.3 && // Second is small
		c3.Close < c3.Open && // Third is bearish
		c3.Close < (c1.Open+c1.Close)/2 // Penetrates first candle
}

// IsThreeWhiteSoldiers checks for three white soldiers pattern
func IsThreeWhiteSoldiers(c1, c2, c3 model.Candle, trend string) bool {
	return trend == "下降" &&
		c1.Close > c1.Open &&
		c2.Close > c2.Open &&
		c3.Close > c3.Open &&
		c2.Open > c1.Open && c2.Open < c1.Close &&
		c3.Open > c2.Open && c3.Open < c2.Close &&
		c2.Close > c1.Close &&
		c3.Close > c2.Close
}

// IsThreeBlackCrows checks for three black crows pattern
func IsThreeBlackCrows(c1, c2, c3 model.Candle, trend string) bool {
	return trend == "上升" &&
		c1.Close < c1.Open &&
		c2.Close < c2.Open &&
		c3.Close < c3.Open &&
		c2.Open < c1.Close && c2.Open > c1.Open &&
		c3.Open < c2.Close && c3.Open > c2.Open &&
		c2.Close < c1.Close &&
		c3.Close < c2.Close
}
