package exchange

import (
	"fmt"
	ccxt "github.com/ccxt/ccxt/go/v4"
	"log"
)

type Binance struct {
	Exchange ccxt.Binance
}

func NewBinance(key string, secret string, proxy string) Exchange {
	exchange := ccxt.NewBinance(map[string]interface{}{
		"apiKey": key,
		"secret": secret,
		"options": map[string]interface{}{
			"defaultType": "future",
		},
	})
	if proxy != "" {
		exchange.WsProxy = proxy
		exchange.HttpsProxy = proxy
	}
	<-exchange.LoadMarkets()
	return &Binance{
		Exchange: exchange,
	}
}

func (m *Binance) FetchPositions() (interface{}, error) {
	positions, err := m.Exchange.FetchPositions()
	if err != nil {
		log.Printf("⚠️ Fetch positions error: %v", err)
		return nil, err
	}
	return positions, nil
}

func (m *Binance) AddMargin(symbol string, amount float64) string {
	marginChan := m.Exchange.AddMargin(symbol, amount)
	result := <-marginChan
	if resultData, ok := result.(map[string]interface{}); ok {
		if status, ok := resultData["status"].(string); ok && status == "ok" {
			msg := fmt.Sprintf("✅ Margin added: %s +%.2f USDT", symbol, amount)
			log.Println(msg)
			return msg
		}
	}

	msg := fmt.Sprintf("❌ Margin add failed: %s +%.2f USDT", symbol, amount)
	println(msg)
	return msg
}

func (m *Binance) GetName() string {
	return "Binance"
}
