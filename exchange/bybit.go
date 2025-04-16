package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	bybit "github.com/bybit-exchange/bybit.go.api"
	"log"
	"margin_monitor/model"
)

type ByBit struct {
	Exchange *bybit.Client
}

func NewByBit(key string, secret string, proxy string) Exchange {
	client := bybit.NewBybitHttpClient(key, secret, bybit.WithBaseURL(bybit.MAINNET), bybit.WithProxyURL(proxy))
	return &ByBit{
		Exchange: client,
	}
}

func (m *ByBit) FetchPositions() (interface{}, error) {
	params := map[string]interface{}{"category": "linear", "settleCoin": "USDT", "limit": 100}
	result, err := m.Exchange.NewUtaBybitServiceWithParams(params).GetPositionList(context.Background())
	if err != nil {
		log.Println("[ByBit] Fetch Positions Error", err.Error())
		return nil, err
	}
	if result.RetCode == 0 && result.RetMsg == "OK" {
		positions, err := mapToStruct[model.PositionList](result.Result)
		if err != nil {
			log.Println("[ByBit] Fetch Positions Error", err.Error())
		}
		return positions, err
	}
	return nil, errors.New("[ByBit] Fetch Positions Error")
}

func (m *ByBit) AddMargin(symbol string, amount float64) string {
	params := map[string]interface{}{
		"symbol":        symbol,
		"category":      "linear",
		"autoAddMargin": 1,
	}
	result, err := m.Exchange.NewUtaBybitServiceWithParams(params).SetPositionAutoMargin(context.Background())
	if err != nil {
		log.Println("[ByBit] Add Margin Error", err.Error())
		return fmt.Sprintf("[ByBit] Add Margin Error: %s", err.Error())
	}
	if result.RetCode == 0 && result.RetMsg == "OK" {
		return "✅ 自动追加保证金设置成功"
	}
	if result.RetCode == 10001 {
		return "⚠️ 自动追加保证金设置未修改（可能已是目标状态）"
	}
	return "❌ 响应解析失败，接口调用异常"
}

func (m *ByBit) GetName() string {
	return "ByBit"
}

func mapToStruct[T any](m interface{}) (*T, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal error: %w", err)
	}
	var result T
	if err := json.Unmarshal(bytes, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}
	return &result, nil
}
