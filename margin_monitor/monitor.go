package margin_monitor

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"margin_monitor/config"
)

type Monitor struct {
	Exchange ccxt.Binance
	TGBot    *tgbotapi.BotAPI
	ChatID   int64
}

func NewMonitor(conf *config.Config) (*Monitor, error) {
	exchange := ccxt.NewBinance(map[string]interface{}{
		"apiKey": conf.Exchange.ExchangeKey,
		"secret": conf.Exchange.ExchangeSecret,
		"options": map[string]interface{}{
			"defaultType": "future",
		},
	})
	exchange.WsProxy = conf.Proxy
	exchange.HttpsProxy = conf.Proxy
	<-exchange.LoadMarkets()

	proxyURL, _ := url.Parse(conf.Proxy)
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	bot, err := tgbotapi.NewBotAPIWithClient(conf.Telegram.BotToken, tgbotapi.APIEndpoint, client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize telegram bot: %w", err)
	}

	return &Monitor{
		Exchange: exchange,
		TGBot:    bot,
		ChatID:   conf.Telegram.ChatID,
	}, nil
}

func (m *Monitor) FetchPositions() ([]ccxt.Position, error) {
	positions, err := m.Exchange.FetchPositions()
	if err != nil {
		m.SendTelegramMessage(fmt.Sprintf("⚠️ Fetch positions error: %v", err))
		return nil, err
	}
	return positions, nil
}

func (m *Monitor) AddMargin(symbol string, amount float64) {
	marginChan := m.Exchange.AddMargin(symbol, amount)
	result := <-marginChan
	if resultData, ok := result.(map[string]interface{}); ok {
		if status, ok := resultData["status"].(string); ok && status == "ok" {
			msg := fmt.Sprintf("✅ Margin added: %s +%.2f USDT", symbol, amount)
			fmt.Println(msg)
			m.SendTelegramMessage(msg)
			return
		}
	}

	msg := fmt.Sprintf("❌ Margin add failed: %s +%.2f USDT", symbol, amount)
	fmt.Println(msg)
	m.SendTelegramMessage(msg)
}

func (m *Monitor) SendTelegramMessage(message string) {
	if m.TGBot == nil {
		log.Println("Telegram bot is not initialized")
		return
	}

	msg := tgbotapi.NewMessage(m.ChatID, message)
	_, err := m.TGBot.Send(msg)
	if err != nil {
		log.Printf("Failed to send Telegram message: %v", err)
	}
}
