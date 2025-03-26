package main

import (
	"context"
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/gin-gonic/gin"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type Candle struct {
	Timestamp int64
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	DateTime  time.Time
}

// Trade represents one completed trade.
type Trade struct {
	Symbol     string  `json:"symbol"`
	EntryTime  string  `json:"entry_time"`
	EntryPrice float64 `json:"entry_price"`
	ExitTime   string  `json:"exit_time"`
	ExitPrice  float64 `json:"exit_price"`
	TradeType  string  `json:"trade_type"`
	ProfitPct  float64 `json:"profit_pct"`
	// Optional RSI values for RSI strategies:
	EntryRSI float64 `json:"entry_rsi,omitempty"`
	ExitRSI  float64 `json:"exit_rsi,omitempty"`
}

// User holds trading parameters.
type User struct {
	Username      string
	RSIPeriod     int
	BuyThreshold  float64
	SellThreshold float64
	TradeType     string // "long" or "short"
	Strategy      string // "RSI", "MA", "RSI_MA", "KAGE", "KITSUNE", "RYU", "SAKURA", "HIKARI", "TENSHI", "ZEN", "RAMSEY"
	MAPeriod      int
}

// ---------------------- Utility Functions ----------------------

// logTrade writes trade details to a log file.
func logTrade(details string) {
	log.Println(details)
}

// loadCSVData loads candle data from a CSV file.
// CSV is expected to have columns: timestamp, open, high, low, close, volume.
func loadCSVData(filepath string) ([]Candle, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	var candles []Candle
	// Skip header row
	_, err = r.Read()
	if err != nil {
		return nil, err
	}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		ts, _ := strconv.ParseInt(record[0], 10, 64)
		open, _ := strconv.ParseFloat(record[1], 64)
		high, _ := strconv.ParseFloat(record[2], 64)
		low, _ := strconv.ParseFloat(record[3], 64)
		closePrice, _ := strconv.ParseFloat(record[4], 64)
		volume, _ := strconv.ParseFloat(record[5], 64)
		// Convert timestamp (in milliseconds) to IST (UTC+5:30)
		dt := time.UnixMilli(ts).In(time.FixedZone("IST", 5*3600+1800))
		candle := Candle{
			Timestamp: ts,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
			DateTime:  dt,
		}
		candles = append(candles, candle)
	}
	return candles, nil
}

// calculateNovelStochastic computes a custom stochastic oscillator over a rolling window.
func calculateNovelStochastic(candles []Candle, period int) []float64 {
	stoch := make([]float64, len(candles))
	for i := range candles {
		start := i - period + 1
		if start < 0 {
			start = 0
		}
		lowMin := candles[start].Low
		highMax := candles[start].High
		for j := start; j <= i; j++ {
			if candles[j].Low < lowMin {
				lowMin = candles[j].Low
			}
			if candles[j].High > highMax {
				highMax = candles[j].High
			}
		}
		rangeVal := highMax - lowMin
		if rangeVal == 0 {
			stoch[i] = 50.0
		} else {
			stoch[i] = ((candles[i].Close - lowMin) / rangeVal) * 100
		}
	}
	return stoch
}

// calculateTradeSummary returns a summary of trades.
func calculateTradeSummary(trades []Trade) map[string]interface{} {
	totalTrades := len(trades)
	winningTrades := 0
	totalProfit := 0.0
	for _, t := range trades {
		totalProfit += t.ProfitPct
		if t.ProfitPct > 0 {
			winningTrades++
		}
	}
	avgProfit := 0.0
	if totalTrades > 0 {
		avgProfit = totalProfit / float64(totalTrades)
	}
	return map[string]interface{}{
		"total_trades":         totalTrades,
		"winning_trades":       winningTrades,
		"total_profit_pct":     totalProfit,
		"avg_profit_per_trade": avgProfit,
	}
}

// generatePlots creates a simple price chart with trade markers and returns a base64-encoded PNG.
// generatePlots creates a simple price chart with trade markers and returns a base64-encoded PNG.
// generatePlots creates a simple price chart with trade markers and returns a base64-encoded PNG.
func generatePlots(candles []Candle, trades []Trade, strategyName string, rsiPeriod, maPeriod int, tradeType string) (string, error) {
	p := plot.New()
	p.Title.Text = fmt.Sprintf("%s Strategy - %s Trades", strategyName, tradeType)
	p.X.Label.Text = "Timestamp"
	p.Y.Label.Text = "Price"

	pts := make(plotter.XYs, len(candles))
	for i, c := range candles {
		pts[i].X = float64(c.Timestamp)
		pts[i].Y = c.Close
	}
	line, err := plotter.NewLine(pts)
	if err != nil {
		return "", err
	}
	p.Add(line)

	// Plot trade markers
	for _, trade := range trades {
		entryTime, _ := time.Parse(time.RFC3339, trade.EntryTime)
		exitTime, _ := time.Parse(time.RFC3339, trade.ExitTime)
		entryX := float64(entryTime.UnixMilli())
		exitX := float64(exitTime.UnixMilli())

		entryPts := plotter.XYs{{X: entryX, Y: trade.EntryPrice}}
		exitPts := plotter.XYs{{X: exitX, Y: trade.ExitPrice}}
		entryScatter, err := plotter.NewScatter(entryPts)
		if err != nil {
			continue
		}
		exitScatter, err := plotter.NewScatter(exitPts)
		if err != nil {
			continue
		}
		p.Add(entryScatter, exitScatter)

		tradeLinePts := plotter.XYs{
			{X: entryX, Y: trade.EntryPrice},
			{X: exitX, Y: trade.ExitPrice},
		}
		tradeLine, err := plotter.NewLine(tradeLinePts)
		if err != nil {
			continue
		}
		tradeLine.Dashes = []vg.Length{vg.Points(5), vg.Points(5)}
		p.Add(tradeLine)
	}

	// Save the plot to a temporary file.
	tmpFile, err := os.CreateTemp("", "plot-*.png")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if err := p.Save(500, 300, tmpFile.Name()); err != nil {
		return "", err
	}
	// Read the file and encode its contents.
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return encoded, nil
}

// ---------------------- TradingStrategy Struct ----------------------

// TradingStrategy encapsulates trading logic and holds a Binance client instance.
type TradingStrategy struct {
	user   User
	client *binance.Client
}

// fetchData retrieves historical klines data from Binance (or loads from CSV if requested).
func (ts *TradingStrategy) fetchData(symbol, interval string, limit int, useCSV bool) ([]Candle, error) {
	if useCSV {
		return loadCSVData("minute_data.csv")
	}
	// Use go-binance to fetch kline data.
	klines, err := ts.client.NewKlinesService().
		Symbol(symbol).
		Interval(interval).
		Limit(limit).
		Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error fetching data for %s: %v", symbol, err)
	}
	var candles []Candle
	for _, k := range klines {
		// Binance returns OpenTime in milliseconds.
		tsVal := k.OpenTime
		open, _ := strconv.ParseFloat(k.Open, 64)
		high, _ := strconv.ParseFloat(k.High, 64)
		low, _ := strconv.ParseFloat(k.Low, 64)
		closePrice, _ := strconv.ParseFloat(k.Close, 64)
		volume, _ := strconv.ParseFloat(k.Volume, 64)
		// Convert timestamp to IST (UTC+5:30)
		dt := time.UnixMilli(tsVal).In(time.FixedZone("IST", 5*3600+1800))
		candle := Candle{
			Timestamp: tsVal,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
			DateTime:  dt,
		}
		candles = append(candles, candle)
	}
	return candles, nil
}

// safeProfitPct calculates the profit percentage safely.
func (ts *TradingStrategy) safeProfitPct(entryPrice, exitPrice float64, tradeType string) float64 {
	if entryPrice == 0 {
		return 0
	}
	if tradeType == "long" {
		return ((exitPrice - entryPrice) / entryPrice) * 100
	}
	return ((entryPrice - exitPrice) / entryPrice) * 100
}

// ---------------------- Strategy Implementations ----------------------

// KAGE strategy: uses volatility and novel stochastic indicator.
func (ts *TradingStrategy) executeKage(symbol string, useCSV bool) (map[string]interface{}, error) {
	candles, err := ts.fetchData(symbol, "1m", 10000, useCSV)
	if err != nil || len(candles) == 0 {
		return map[string]interface{}{"error": fmt.Sprintf("No data fetched for %s", symbol)}, err
	}
	stoch := calculateNovelStochastic(candles, 14)

	// Compute log returns.
	returns := make([]float64, len(candles))
	for i := 1; i < len(candles); i++ {
		returns[i] = math.Log(candles[i].Close / candles[i-1].Close)
	}
	// Rolling volatility (window=30)
	window := 30
	vol := make([]float64, len(candles))
	for i := window; i < len(candles); i++ {
		sum := 0.0
		for j := i - window; j < i; j++ {
			sum += returns[j]
		}
		mean := sum / float64(window)
		sumSq := 0.0
		for j := i - window; j < i; j++ {
			sumSq += (returns[j] - mean) * (returns[j] - mean)
		}
		vol[i] = math.Sqrt(sumSq / float64(window))
	}
	// Volatility threshold
	volSum, countVol := 0.0, 0
	for _, v := range vol {
		if v != 0 {
			volSum += v
			countVol++
		}
	}
	meanVol := 0.0
	if countVol > 0 {
		meanVol = volSum / float64(countVol)
	}
	thresholdVol := meanVol * 1.5

	var trades []Trade
	var openPositions []struct {
		EntryTime  time.Time
		EntryPrice float64
	}

	for i := window; i < len(candles); i++ {
		currentTime := candles[i].DateTime
		currentPrice := candles[i].Close
		currentVol := vol[i]
		stochVal := stoch[i]
		if ts.user.TradeType == "long" {
			if currentVol < thresholdVol && stochVal < 20 && len(openPositions) == 0 {
				openPositions = append(openPositions, struct {
					EntryTime  time.Time
					EntryPrice float64
				}{currentTime, currentPrice})
			} else if currentVol > thresholdVol && stochVal > 80 && len(openPositions) > 0 {
				pos := openPositions[0]
				openPositions = openPositions[1:]
				profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "long")
				trade := Trade{
					Symbol:     symbol,
					EntryTime:  pos.EntryTime.Format(time.RFC3339),
					EntryPrice: pos.EntryPrice,
					ExitTime:   currentTime.Format(time.RFC3339),
					ExitPrice:  currentPrice,
					TradeType:  "long",
					ProfitPct:  profit,
				}
				trades = append(trades, trade)
				logTrade(fmt.Sprintf("KAGE Long trade for %s: Buy at %s (price: %.4f) | Sell at %s (price: %.4f) | P/L: %.2f%%",
					symbol, pos.EntryTime.Format(time.RFC3339), pos.EntryPrice, currentTime.Format(time.RFC3339), currentPrice, profit))
			}
		} else {
			if currentVol < thresholdVol && stochVal > 80 && len(openPositions) == 0 {
				openPositions = append(openPositions, struct {
					EntryTime  time.Time
					EntryPrice float64
				}{currentTime, currentPrice})
			} else if currentVol > thresholdVol && stochVal < 20 && len(openPositions) > 0 {
				pos := openPositions[0]
				openPositions = openPositions[1:]
				profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "short")
				trade := Trade{
					Symbol:     symbol,
					EntryTime:  pos.EntryTime.Format(time.RFC3339),
					EntryPrice: pos.EntryPrice,
					ExitTime:   currentTime.Format(time.RFC3339),
					ExitPrice:  currentPrice,
					TradeType:  "short",
					ProfitPct:  profit,
				}
				trades = append(trades, trade)
				logTrade(fmt.Sprintf("KAGE Short trade for %s: Sell at %s (price: %.4f) | Cover at %s (price: %.4f) | P/L: %.2f%%",
					symbol, pos.EntryTime.Format(time.RFC3339), pos.EntryPrice, currentTime.Format(time.RFC3339), currentPrice, profit))
			}
		}
	}

	summary := calculateTradeSummary(trades)
	plotImage, err := generatePlots(candles, trades, "KAGE", ts.user.RSIPeriod, ts.user.MAPeriod, ts.user.TradeType)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"trades":  trades,
		"plot":    plotImage,
		"summary": summary,
	}, nil
}

// KITSUNE strategy: uses rolling z-score of prices and novel stochastic indicator.
func (ts *TradingStrategy) executeKitsune(symbol string, useCSV bool) (map[string]interface{}, error) {
	candles, err := ts.fetchData(symbol, "1m", 1000, useCSV)
	if err != nil || len(candles) == 0 {
		return map[string]interface{}{"error": fmt.Sprintf("No data fetched for %s", symbol)}, err
	}
	stoch := calculateNovelStochastic(candles, 14)
	window := 20
	var trades []Trade
	var openPositions []struct {
		EntryTime  time.Time
		EntryPrice float64
	}

	for i := window; i < len(candles); i++ {
		// Compute z-score of close price over a rolling window.
		sum := 0.0
		for j := i - window; j < i; j++ {
			sum += candles[j].Close
		}
		mean := sum / float64(window)
		variance := 0.0
		for j := i - window; j < i; j++ {
			variance += (candles[j].Close - mean) * (candles[j].Close - mean)
		}
		std := math.Sqrt(variance / float64(window))
		if std == 0 {
			std = 1e-8
		}
		zScore := (candles[i].Close - mean) / std
		stochVal := stoch[i]
		currentTime := candles[i].DateTime
		currentPrice := candles[i].Close

		if ts.user.TradeType == "long" {
			if zScore < -1.0 && stochVal < 20 && len(openPositions) == 0 {
				openPositions = append(openPositions, struct {
					EntryTime  time.Time
					EntryPrice float64
				}{currentTime, currentPrice})
			} else if zScore > 1.0 && stochVal > 80 && len(openPositions) > 0 {
				pos := openPositions[0]
				openPositions = openPositions[1:]
				profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "long")
				trade := Trade{
					Symbol:     symbol,
					EntryTime:  pos.EntryTime.Format(time.RFC3339),
					EntryPrice: pos.EntryPrice,
					ExitTime:   currentTime.Format(time.RFC3339),
					ExitPrice:  currentPrice,
					TradeType:  "long",
					ProfitPct:  profit,
				}
				trades = append(trades, trade)
				logTrade(fmt.Sprintf("KITSUNE Long trade for %s: Buy at %s | Sell at %s | P/L: %.2f%%",
					symbol, pos.EntryTime.Format(time.RFC3339), currentTime.Format(time.RFC3339), profit))
			}
		} else {
			if zScore > 1.0 && stochVal > 80 && len(openPositions) == 0 {
				openPositions = append(openPositions, struct {
					EntryTime  time.Time
					EntryPrice float64
				}{currentTime, currentPrice})
			} else if zScore < -1.0 && stochVal < 20 && len(openPositions) > 0 {
				pos := openPositions[0]
				openPositions = openPositions[1:]
				profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "short")
				trade := Trade{
					Symbol:     symbol,
					EntryTime:  pos.EntryTime.Format(time.RFC3339),
					EntryPrice: pos.EntryPrice,
					ExitTime:   currentTime.Format(time.RFC3339),
					ExitPrice:  currentPrice,
					TradeType:  "short",
					ProfitPct:  profit,
				}
				trades = append(trades, trade)
				logTrade(fmt.Sprintf("KITSUNE Short trade for %s: Sell at %s | Cover at %s | P/L: %.2f%%",
					symbol, pos.EntryTime.Format(time.RFC3339), currentTime.Format(time.RFC3339), profit))
			}
		}
	}

	summary := calculateTradeSummary(trades)
	plotImage, err := generatePlots(candles, trades, "KITSUNE", ts.user.RSIPeriod, ts.user.MAPeriod, ts.user.TradeType)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"trades":  trades,
		"plot":    plotImage,
		"summary": summary,
	}, nil
}

// RYU strategy: uses z-score of logarithmic returns.
func (ts *TradingStrategy) executeRyu(symbol string, useCSV bool) (map[string]interface{}, error) {
	candles, err := ts.fetchData(symbol, "1m", 1000, useCSV)
	if err != nil || len(candles) < 51 {
		return map[string]interface{}{"error": fmt.Sprintf("No data fetched for %s", symbol)}, err
	}
	stoch := calculateNovelStochastic(candles, 14)
	// Compute log returns.
	returns := make([]float64, len(candles))
	for i := 1; i < len(candles); i++ {
		returns[i] = math.Log(candles[i].Close / candles[i-1].Close)
	}
	window := 50
	var trades []Trade
	var openPositions []struct {
		EntryTime  time.Time
		EntryPrice float64
	}
	for i := window; i < len(candles); i++ {
		sum := 0.0
		for j := i - window; j < i; j++ {
			sum += returns[j]
		}
		mean := sum / float64(window)
		variance := 0.0
		for j := i - window; j < i; j++ {
			variance += (returns[j] - mean) * (returns[j] - mean)
		}
		std := math.Sqrt(variance / float64(window))
		if std == 0 {
			std = 1e-8
		}
		zReturn := (returns[i] - mean) / std
		stochVal := stoch[i]
		currentTime := candles[i].DateTime
		currentPrice := candles[i].Close

		if ts.user.TradeType == "long" {
			if zReturn < -1 && stochVal < 20 && len(openPositions) == 0 {
				openPositions = append(openPositions, struct {
					EntryTime  time.Time
					EntryPrice float64
				}{currentTime, currentPrice})
			} else if zReturn > 1 && stochVal > 80 && len(openPositions) > 0 {
				pos := openPositions[0]
				openPositions = openPositions[1:]
				profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "long")
				trade := Trade{
					Symbol:     symbol,
					EntryTime:  pos.EntryTime.Format(time.RFC3339),
					EntryPrice: pos.EntryPrice,
					ExitTime:   currentTime.Format(time.RFC3339),
					ExitPrice:  currentPrice,
					TradeType:  "long",
					ProfitPct:  profit,
				}
				trades = append(trades, trade)
				logTrade(fmt.Sprintf("RYU Long trade for %s: Buy at %s | Sell at %s | P/L: %.2f%%",
					symbol, pos.EntryTime.Format(time.RFC3339), currentTime.Format(time.RFC3339), profit))
			}
		} else {
			if zReturn > 1 && stochVal > 80 && len(openPositions) == 0 {
				openPositions = append(openPositions, struct {
					EntryTime  time.Time
					EntryPrice float64
				}{currentTime, currentPrice})
			} else if zReturn < -1 && stochVal < 20 && len(openPositions) > 0 {
				pos := openPositions[0]
				openPositions = openPositions[1:]
				profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "short")
				trade := Trade{
					Symbol:     symbol,
					EntryTime:  pos.EntryTime.Format(time.RFC3339),
					EntryPrice: pos.EntryPrice,
					ExitTime:   currentTime.Format(time.RFC3339),
					ExitPrice:  currentPrice,
					TradeType:  "short",
					ProfitPct:  profit,
				}
				trades = append(trades, trade)
				logTrade(fmt.Sprintf("RYU Short trade for %s: Sell at %s | Cover at %s | P/L: %.2f%%",
					symbol, pos.EntryTime.Format(time.RFC3339), currentTime.Format(time.RFC3339), profit))
			}
		}
	}
	summary := calculateTradeSummary(trades)
	plotImage, err := generatePlots(candles, trades, "RYU", ts.user.RSIPeriod, ts.user.MAPeriod, ts.user.TradeType)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"trades":  trades,
		"plot":    plotImage,
		"summary": summary,
	}, nil
}

// SAKURA strategy: uses regression on segments with novel stochastic confirmation.
func (ts *TradingStrategy) executeSakura(symbol string, useCSV bool) (map[string]interface{}, error) {
	candles, err := ts.fetchData(symbol, "1m", 1000, useCSV)
	if err != nil || len(candles) < 50 {
		return map[string]interface{}{"error": fmt.Sprintf("No data fetched for %s", symbol)}, err
	}
	stoch := calculateNovelStochastic(candles, 14)
	window := 50
	var trades []Trade
	var openPositions []struct {
		EntryTime  time.Time
		EntryPrice float64
	}
	// Helper: median of a slice of float64.
	median := func(arr []float64) float64 {
		sorted := append([]float64{}, arr...)
		sort.Float64s(sorted)
		mid := len(sorted) / 2
		if len(sorted)%2 == 0 {
			return (sorted[mid-1] + sorted[mid]) / 2
		}
		return sorted[mid]
	}
	// Simple linear regression: returns slope.
	regressionSlope := func(x, y []float64) float64 {
		n := float64(len(x))
		sumX, sumY, sumXY, sumXX := 0.0, 0.0, 0.0, 0.0
		for i := 0; i < len(x); i++ {
			sumX += x[i]
			sumY += y[i]
			sumXY += x[i] * y[i]
			sumXX += x[i] * x[i]
		}
		denom := n*sumXX - sumX*sumX
		if denom == 0 {
			return 0
		}
		return (n*sumXY - sumX*sumY) / denom
	}

	for i := window; i < len(candles); i++ {
		// Get last window of close prices.
		var prices []float64
		for j := i - window; j < i; j++ {
			prices = append(prices, candles[j].Close)
		}
		pivot := median(prices)
		var upSegment, downSegment []float64
		for _, p := range prices {
			if p > pivot {
				upSegment = append(upSegment, p)
			} else {
				downSegment = append(downSegment, p)
			}
		}
		stochVal := stoch[i]
		currentTime := candles[i].DateTime
		currentPrice := candles[i].Close
		if len(upSegment) >= 2 && len(downSegment) >= 2 {
			// Prepare x-values for regression.
			xUp := make([]float64, len(upSegment))
			for idx := range upSegment {
				xUp[idx] = float64(idx)
			}
			slopeUp := regressionSlope(xUp, upSegment)

			xDown := make([]float64, len(downSegment))
			for idx := range downSegment {
				xDown[idx] = float64(idx)
			}
			slopeDown := regressionSlope(xDown, downSegment)

			slope := (slopeUp - slopeDown) / 2.0
			mirrorPrice := (upSegment[len(upSegment)-1]+downSegment[len(downSegment)-1])/2.0 + slope
			deviation := math.Abs(currentPrice - mirrorPrice)
			threshold := currentPrice * 0.003
			if ts.user.TradeType == "long" {
				if deviation < threshold && stochVal < 20 && len(openPositions) == 0 {
					openPositions = append(openPositions, struct {
						EntryTime  time.Time
						EntryPrice float64
					}{currentTime, currentPrice})
				} else if deviation > threshold && stochVal > 80 && len(openPositions) > 0 {
					pos := openPositions[0]
					openPositions = openPositions[1:]
					profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "long")
					trade := Trade{
						Symbol:     symbol,
						EntryTime:  pos.EntryTime.Format(time.RFC3339),
						EntryPrice: pos.EntryPrice,
						ExitTime:   currentTime.Format(time.RFC3339),
						ExitPrice:  currentPrice,
						TradeType:  "long",
						ProfitPct:  profit,
					}
					trades = append(trades, trade)
					logTrade(fmt.Sprintf("SAKURA Long trade for %s: Buy at %s | Sell at %s | P/L: %.2f%%",
						symbol, pos.EntryTime.Format(time.RFC3339), currentTime.Format(time.RFC3339), profit))
				}
			} else {
				if deviation < threshold && stochVal > 80 && len(openPositions) == 0 {
					openPositions = append(openPositions, struct {
						EntryTime  time.Time
						EntryPrice float64
					}{currentTime, currentPrice})
				} else if deviation > threshold && stochVal < 20 && len(openPositions) > 0 {
					pos := openPositions[0]
					openPositions = openPositions[1:]
					profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "short")
					trade := Trade{
						Symbol:     symbol,
						EntryTime:  pos.EntryTime.Format(time.RFC3339),
						EntryPrice: pos.EntryPrice,
						ExitTime:   currentTime.Format(time.RFC3339),
						ExitPrice:  currentPrice,
						TradeType:  "short",
						ProfitPct:  profit,
					}
					trades = append(trades, trade)
					logTrade(fmt.Sprintf("SAKURA Short trade for %s: Sell at %s | Cover at %s | P/L: %.2f%%",
						symbol, pos.EntryTime.Format(time.RFC3339), currentTime.Format(time.RFC3339), profit))
				}
			}
		}
	}
	summary := calculateTradeSummary(trades)
	plotImage, err := generatePlots(candles, trades, "SAKURA", ts.user.RSIPeriod, ts.user.MAPeriod, ts.user.TradeType)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"trades":  trades,
		"plot":    plotImage,
		"summary": summary,
	}, nil
}

// HIKARI strategy: uses a simplified PCA-like approach on returns.
func (ts *TradingStrategy) executeHikari(symbol string, useCSV bool) (map[string]interface{}, error) {
	candles, err := ts.fetchData(symbol, "1m", 1000, useCSV)
	if err != nil || len(candles) < 31 {
		return map[string]interface{}{"error": fmt.Sprintf("No data fetched for %s", symbol)}, err
	}
	stoch := calculateNovelStochastic(candles, 14)
	// Compute log returns.
	returns := make([]float64, len(candles))
	for i := 1; i < len(candles); i++ {
		returns[i] = math.Log(candles[i].Close / candles[i-1].Close)
	}
	window := 30
	var trades []Trade
	var openPositions []struct {
		EntryTime  time.Time
		EntryPrice float64
	}
	for i := window; i < len(candles); i++ {
		// For simplicity, use current return as momentum.
		currentReturn := returns[i]
		momentum := currentReturn // In practice, you might perform PCA on window returns.
		stochVal := stoch[i]
		currentTime := candles[i].DateTime
		currentPrice := candles[i].Close

		if ts.user.TradeType == "long" {
			if momentum > 0.0005 && stochVal < 20 && len(openPositions) == 0 {
				openPositions = append(openPositions, struct {
					EntryTime  time.Time
					EntryPrice float64
				}{currentTime, currentPrice})
			} else if momentum < 0 && stochVal > 80 && len(openPositions) > 0 {
				pos := openPositions[0]
				openPositions = openPositions[1:]
				profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "long")
				trade := Trade{
					Symbol:     symbol,
					EntryTime:  pos.EntryTime.Format(time.RFC3339),
					EntryPrice: pos.EntryPrice,
					ExitTime:   currentTime.Format(time.RFC3339),
					ExitPrice:  currentPrice,
					TradeType:  "long",
					ProfitPct:  profit,
				}
				trades = append(trades, trade)
				logTrade(fmt.Sprintf("HIKARI Long trade for %s: Buy at %s | Sell at %s | P/L: %.2f%%",
					symbol, pos.EntryTime.Format(time.RFC3339), currentTime.Format(time.RFC3339), profit))
			}
		} else {
			if momentum < -0.0005 && stochVal > 80 && len(openPositions) == 0 {
				openPositions = append(openPositions, struct {
					EntryTime  time.Time
					EntryPrice float64
				}{currentTime, currentPrice})
			} else if momentum > 0 && stochVal < 20 && len(openPositions) > 0 {
				pos := openPositions[0]
				openPositions = openPositions[1:]
				profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "short")
				trade := Trade{
					Symbol:     symbol,
					EntryTime:  pos.EntryTime.Format(time.RFC3339),
					EntryPrice: pos.EntryPrice,
					ExitTime:   currentTime.Format(time.RFC3339),
					ExitPrice:  currentPrice,
					TradeType:  "short",
					ProfitPct:  profit,
				}
				trades = append(trades, trade)
				logTrade(fmt.Sprintf("HIKARI Short trade for %s: Sell at %s | Cover at %s | P/L: %.2f%%",
					symbol, pos.EntryTime.Format(time.RFC3339), currentTime.Format(time.RFC3339), profit))
			}
		}
	}
	summary := calculateTradeSummary(trades)
	plotImage, err := generatePlots(candles, trades, "HIKARI", ts.user.RSIPeriod, ts.user.MAPeriod, ts.user.TradeType)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"trades":  trades,
		"plot":    plotImage,
		"summary": summary,
	}, nil
}

// TENSHI strategy: uses local-extrema detection with novel stochastic confirmation.
func (ts *TradingStrategy) executeTenshi(symbol string, useCSV bool) (map[string]interface{}, error) {
	candles, err := ts.fetchData(symbol, "1m", 1000, useCSV)
	if err != nil || len(candles) < 3 {
		return map[string]interface{}{"error": fmt.Sprintf("No data fetched for %s", symbol)}, err
	}
	stoch := calculateNovelStochastic(candles, 14)
	var trades []Trade
	var openPositions []struct {
		EntryTime  time.Time
		EntryPrice float64
	}
	// Detect local extrema: iterate from index 1 to len-2.
	for i := 1; i < len(candles)-1; i++ {
		currentTime := candles[i].DateTime
		currentPrice := candles[i].Close
		stochVal := stoch[i]
		// Local minimum
		if candles[i].Close < candles[i-1].Close && candles[i].Close < candles[i+1].Close {
			if ts.user.TradeType == "long" && stochVal < 20 && len(openPositions) == 0 {
				openPositions = append(openPositions, struct {
					EntryTime  time.Time
					EntryPrice float64
				}{currentTime, currentPrice})
			}
		}
		// Local maximum
		if candles[i].Close > candles[i-1].Close && candles[i].Close > candles[i+1].Close {
			if ts.user.TradeType == "long" && stochVal > 80 && len(openPositions) > 0 {
				pos := openPositions[0]
				openPositions = openPositions[1:]
				profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "long")
				trade := Trade{
					Symbol:     symbol,
					EntryTime:  pos.EntryTime.Format(time.RFC3339),
					EntryPrice: pos.EntryPrice,
					ExitTime:   currentTime.Format(time.RFC3339),
					ExitPrice:  currentPrice,
					TradeType:  "long",
					ProfitPct:  profit,
				}
				trades = append(trades, trade)
				logTrade(fmt.Sprintf("TENSHI Long trade for %s: Buy at %s | Sell at %s | P/L: %.2f%%",
					symbol, pos.EntryTime.Format(time.RFC3339), currentTime.Format(time.RFC3339), profit))
			}
		}
	}
	summary := calculateTradeSummary(trades)
	plotImage, err := generatePlots(candles, trades, "TENSHI", ts.user.RSIPeriod, ts.user.MAPeriod, ts.user.TradeType)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"trades":  trades,
		"plot":    plotImage,
		"summary": summary,
	}, nil
}

// ZEN strategy: uses Bollinger Bands, normalized phase and momentum.
func (ts *TradingStrategy) executeZen(symbol string, useCSV bool) (map[string]interface{}, error) {
	candles, err := ts.fetchData(symbol, "1m", 1000, useCSV)
	if err != nil || len(candles) < 21 {
		return map[string]interface{}{"error": fmt.Sprintf("No data fetched for %s", symbol)}, err
	}
	stoch := calculateNovelStochastic(candles, 14)
	window := 20
	var trades []Trade
	var openPositions []struct {
		EntryTime  time.Time
		EntryPrice float64
	}
	// Pre-calculate SMA and standard deviation for Bollinger Bands.
	sma := make([]float64, len(candles))
	stdDev := make([]float64, len(candles))
	for i := range candles {
		if i < window-1 {
			sma[i] = 0
			stdDev[i] = 0
		} else {
			sum := 0.0
			for j := i - window + 1; j <= i; j++ {
				sum += candles[j].Close
			}
			mean := sum / float64(window)
			sma[i] = mean
			var sumSq float64
			for j := i - window + 1; j <= i; j++ {
				sumSq += (candles[j].Close - mean) * (candles[j].Close - mean)
			}
			stdDev[i] = math.Sqrt(sumSq / float64(window))
		}
	}
	// Calculate normalized phase and momentum.
	normalizedPhase := make([]float64, len(candles))
	momentum := make([]float64, len(candles))
	for i := 0; i < len(candles); i++ {
		if i < window-1 {
			normalizedPhase[i] = 0.5
		} else {
			upperBand := sma[i] + 2*stdDev[i]
			lowerBand := sma[i] - 2*stdDev[i]
			if upperBand-lowerBand == 0 {
				normalizedPhase[i] = 0.5
			} else {
				normalizedPhase[i] = (candles[i].Close - lowerBand) / (upperBand - lowerBand)
			}
		}
		if i == 0 {
			momentum[i] = 0
		} else {
			momentum[i] = candles[i].Close - candles[i-1].Close
		}
	}
	for i := window; i < len(candles); i++ {
		phase := normalizedPhase[i]
		stochVal := stoch[i]
		mom := momentum[i]
		currentTime := candles[i].DateTime
		currentPrice := candles[i].Close

		if ts.user.TradeType == "long" {
			if phase < 0.3 && stochVal < 20 && mom > 0 && len(openPositions) == 0 {
				openPositions = append(openPositions, struct {
					EntryTime  time.Time
					EntryPrice float64
				}{currentTime, currentPrice})
			} else if phase > 0.7 && stochVal > 80 && len(openPositions) > 0 {
				pos := openPositions[0]
				openPositions = openPositions[1:]
				profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "long")
				trade := Trade{
					Symbol:     symbol,
					EntryTime:  pos.EntryTime.Format(time.RFC3339),
					EntryPrice: pos.EntryPrice,
					ExitTime:   currentTime.Format(time.RFC3339),
					ExitPrice:  currentPrice,
					TradeType:  "long",
					ProfitPct:  profit,
				}
				trades = append(trades, trade)
				logTrade(fmt.Sprintf("ZEN Long trade for %s: Buy at %s | Sell at %s | P/L: %.2f%%",
					symbol, pos.EntryTime.Format(time.RFC3339), currentTime.Format(time.RFC3339), profit))
			}
		} else {
			if phase > 0.7 && stochVal > 80 && mom < 0 && len(openPositions) == 0 {
				openPositions = append(openPositions, struct {
					EntryTime  time.Time
					EntryPrice float64
				}{currentTime, currentPrice})
			} else if phase < 0.3 && stochVal < 20 && len(openPositions) > 0 {
				pos := openPositions[0]
				openPositions = openPositions[1:]
				profit := ts.safeProfitPct(pos.EntryPrice, currentPrice, "short")
				trade := Trade{
					Symbol:     symbol,
					EntryTime:  pos.EntryTime.Format(time.RFC3339),
					EntryPrice: pos.EntryPrice,
					ExitTime:   currentTime.Format(time.RFC3339),
					ExitPrice:  currentPrice,
					TradeType:  "short",
					ProfitPct:  profit,
				}
				trades = append(trades, trade)
				logTrade(fmt.Sprintf("ZEN Short trade for %s: Sell at %s | Cover at %s | P/L: %.2f%%",
					symbol, pos.EntryTime.Format(time.RFC3339), currentTime.Format(time.RFC3339), profit))
			}
		}
	}
	summary := calculateTradeSummary(trades)
	plotImage, err := generatePlots(candles, trades, "ZEN", ts.user.RSIPeriod, ts.user.MAPeriod, ts.user.TradeType)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"trades":  trades,
		"plot":    plotImage,
		"summary": summary,
	}, nil
}

// RAMSEY strategy: a simplified implementation based on Ramsey numbers for systemic risk.
func (ts *TradingStrategy) executeRamsey(symbol string, useCSV bool) (map[string]interface{}, error) {
	// For the Ramsey strategy we simulate a multi-asset analysis.
	// In practice you would load multiple assets’ time series and compute correlations.
	// Here we simulate with N assets and a random correlation matrix.
	N := 10
	correlationThreshold := 0.7
	targetCliqueSize := 4
	targetIndependentSize := 4

	// Generate a random symmetric correlation matrix.
	matrix := make([][]float64, N)
	for i := 0; i < N; i++ {
		matrix[i] = make([]float64, N)
		for j := 0; j < N; j++ {
			if i == j {
				matrix[i][j] = 1.0
			} else if j < i {
				matrix[i][j] = matrix[j][i]
			} else {
				matrix[i][j] = rand.Float64()
			}
		}
	}
	// Build graph: edge exists if correlation >= threshold.
	adj := make([][]bool, N)
	for i := 0; i < N; i++ {
		adj[i] = make([]bool, N)
		for j := 0; j < N; j++ {
			if i != j && matrix[i][j] >= correlationThreshold {
				adj[i][j] = true
			} else {
				adj[i][j] = false
			}
		}
	}
	// Bron–Kerbosch algorithm to find maximum clique.
	maxClique := []int{}
	var bronKerbosch func(r, p, x []int)
	bronKerbosch = func(r, p, x []int) {
		if len(p) == 0 && len(x) == 0 {
			if len(r) > len(maxClique) {
				cp := make([]int, len(r))
				copy(cp, r)
				maxClique = cp
			}
			return
		}
		for i := 0; i < len(p); i++ {
			v := p[i]
			// Compute neighbors of v.
			var nv []int
			for j := 0; j < N; j++ {
				if adj[v][j] {
					nv = append(nv, j)
				}
			}
			// Intersection of p and nv.
			var pNew []int
			for _, w := range p {
				for _, u := range nv {
					if w == u {
						pNew = append(pNew, w)
						break
					}
				}
			}
			// Intersection of x and nv.
			var xNew []int
			for _, w := range x {
				for _, u := range nv {
					if w == u {
						xNew = append(xNew, w)
						break
					}
				}
			}
			bronKerbosch(append(r, v), pNew, xNew)
			// Remove v from p and add to x.
			p = append(p[:i], p[i+1:]...)
			x = append(x, v)
			i--
		}
	}
	// Initial call: r empty, p = all vertices, x empty.
	allVertices := make([]int, N)
	for i := 0; i < N; i++ {
		allVertices[i] = i
	}
	bronKerbosch([]int{}, allVertices, []int{})
	maxCliqueSize := len(maxClique)

	// For independent set, use complement graph.
	adjComplement := make([][]bool, N)
	for i := 0; i < N; i++ {
		adjComplement[i] = make([]bool, N)
		for j := 0; j < N; j++ {
			if i != j && !adj[i][j] {
				adjComplement[i][j] = true
			} else {
				adjComplement[i][j] = false
			}
		}
	}
	// Reuse Bron-Kerbosch on complement graph.
	maxIndependent := []int{}
	var bronKerboschComp func(r, p, x []int)
	bronKerboschComp = func(r, p, x []int) {
		if len(p) == 0 && len(x) == 0 {
			if len(r) > len(maxIndependent) {
				cp := make([]int, len(r))
				copy(cp, r)
				maxIndependent = cp
			}
			return
		}
		for i := 0; i < len(p); i++ {
			v := p[i]
			var nv []int
			for j := 0; j < N; j++ {
				if adjComplement[v][j] {
					nv = append(nv, j)
				}
			}
			var pNew []int
			for _, w := range p {
				for _, u := range nv {
					if w == u {
						pNew = append(pNew, w)
						break
					}
				}
			}
			var xNew []int
			for _, w := range x {
				for _, u := range nv {
					if w == u {
						xNew = append(xNew, w)
						break
					}
				}
			}
			bronKerboschComp(append(r, v), pNew, xNew)
			p = append(p[:i], p[i+1:]...)
			x = append(x, v)
			i--
		}
	}
	bronKerboschComp([]int{}, allVertices, []int{})
	maxIndependentSize := len(maxIndependent)

	// Generate signal.
	signal := "Neutral"
	if maxCliqueSize >= targetCliqueSize {
		signal = "High Systemic Risk (risk-off)"
	} else if maxIndependentSize >= targetIndependentSize {
		signal = "Arbitrage/Diversification Opportunity (risk-on)"
	}

	// No trades are simulated here; just a signal summary.
	result := map[string]interface{}{
		"signal":               signal,
		"max_clique_size":      maxCliqueSize,
		"max_independent_size": maxIndependentSize,
		"correlation_matrix":   matrix, // for reference
	}
	return result, nil
}

// executeStrategy dispatches the chosen strategy.
func (ts *TradingStrategy) executeStrategy(symbol string, useScratchRSI bool, useCSV bool) (map[string]interface{}, error) {
	switch ts.user.Strategy {
	case "RSI", "MA", "RSI_MA":
		// RSI-based strategies not fully implemented in this Go example.
		return map[string]interface{}{"error": "RSI/MA-based strategies not implemented in this Go example."}, nil
	case "KAGE":
		return ts.executeKage(symbol, useCSV)
	case "KITSUNE":
		return ts.executeKitsune(symbol, useCSV)
	case "RYU":
		return ts.executeRyu(symbol, useCSV)
	case "SAKURA":
		return ts.executeSakura(symbol, useCSV)
	case "HIKARI":
		return ts.executeHikari(symbol, useCSV)
	case "TENSHI":
		return ts.executeTenshi(symbol, useCSV)
	case "ZEN":
		return ts.executeZen(symbol, useCSV)
	case "RAMSEY":
		return ts.executeRamsey(symbol, useCSV)
	default:
		return map[string]interface{}{"error": "Unknown strategy specified."}, nil
	}
}

// ---------------------- HTTP Endpoints ----------------------

func main() {
	// Set up logging to file.
	logFile, err := os.OpenFile("trades.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	router := gin.Default()

	// Create a Binance client (using empty strings for public data).
	binanceClient := binance.NewClient("", "")

	// POST /trade endpoint.
	router.POST("/trade", func(c *gin.Context) {
		var req struct {
			Exchange      string  `json:"exchange"`
			Symbol        string  `json:"symbol"`
			Username      string  `json:"username"`
			RSIPeriod     int     `json:"rsi_period"`
			BuyThreshold  float64 `json:"buy_threshold"`
			SellThreshold float64 `json:"sell_threshold"`
			TradeType     string  `json:"trade_type"`
			Strategy      string  `json:"strategy"`
			MAPeriod      int     `json:"ma_period"`
			UseScratchRSI bool    `json:"use_scratch_rsi"`
			UseCSV        bool    `json:"use_csv"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// Set defaults.
		if req.Exchange == "" {
			req.Exchange = "binance"
		}
		if req.Username == "" {
			req.Username = "default_user"
		}
		if req.RSIPeriod == 0 {
			req.RSIPeriod = 14
		}
		if req.BuyThreshold == 0 {
			req.BuyThreshold = 30
		}
		if req.SellThreshold == 0 {
			req.SellThreshold = 70
		}
		if req.TradeType == "" {
			req.TradeType = "long"
		}
		if req.Strategy == "" {
			req.Strategy = "RSI"
		}
		if req.MAPeriod == 0 {
			req.MAPeriod = 20
		}

		user := User{
			Username:      req.Username,
			RSIPeriod:     req.RSIPeriod,
			BuyThreshold:  req.BuyThreshold,
			SellThreshold: req.SellThreshold,
			TradeType:     req.TradeType,
			Strategy:      req.Strategy,
			MAPeriod:      req.MAPeriod,
		}

		// For this implementation only "binance" is supported.
		if req.Exchange != "binance" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Only 'binance' exchange is supported in this implementation."})
			return
		}

		strategyObj := TradingStrategy{
			user:   user,
			client: binanceClient,
		}
		result, err := strategyObj.executeStrategy(req.Symbol, req.UseScratchRSI, req.UseCSV)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})

	// GET /exchanges endpoint.
	router.GET("/exchanges", func(c *gin.Context) {
		// Only Binance is supported.
		c.JSON(http.StatusOK, gin.H{"exchanges": []string{"binance"}})
	})

	// GET /symbols endpoint.
	router.GET("/symbols", func(c *gin.Context) {
		exchangeID := c.Query("exchange")
		if exchangeID == "" || exchangeID != "binance" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide exchange=binance as a parameter."})
			return
		}
		// Fetch exchange info from Binance.
		exInfo, err := binanceClient.NewExchangeInfoService().Do(context.Background())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		var symbols []string
		for _, sym := range exInfo.Symbols {
			symbols = append(symbols, sym.Symbol)
		}
		c.JSON(http.StatusOK, gin.H{"exchange": "binance", "symbols": symbols})
	})

	// Run server on port 8080.
	router.Run(":8080")
}
