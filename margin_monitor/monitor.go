package margin_monitor

import (
	"fmt"
	"log"
	"margin_monitor/exchange"
	"net/http"
	"net/url"
	"time"

	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"margin_monitor/config"
)

type Monitor struct {
	Exchange []exchange.Exchange
	TGBot    *tgbotapi.BotAPI
	ChatID   int64
}

func NewMonitor(conf *config.Config) (*Monitor, error) {
	ecs := make([]exchange.Exchange, 0)
	for i := range conf.Exchange {
		switch conf.Exchange[i].Name {
		case "bybit":
			ecs = append(ecs, exchange.NewByBit(conf.Exchange[i].ExchangeKey, conf.Exchange[i].ExchangeSecret, conf.Proxy))
		case "binance":
			ecs = append(ecs, exchange.NewBinance(conf.Exchange[i].ExchangeKey, conf.Exchange[i].ExchangeSecret, conf.Proxy))
		}
	}

	proxyURL, _ := url.Parse(conf.Proxy)
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   3 * time.Second,
	}

	bot, err := tgbotapi.NewBotAPIWithClient(conf.Telegram.BotToken, tgbotapi.APIEndpoint, client)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize telegram bot: %w", err)
	}

	return &Monitor{
		Exchange: ecs,
		TGBot:    bot,
		ChatID:   conf.Telegram.ChatID,
	}, nil
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
