package margin_monitor

import (
	ccxt "github.com/ccxt/ccxt/go/v4"
	"margin_monitor/config"
)

type Monitor struct {
	Exchange ccxt.Binance
}

func NewMonitor(conf *config.Config) (*Monitor, error) {
	exchange := ccxt.NewBinance(map[string]interface{}{
		"apiKey": conf.Exchange.ExchangeKey,
		"secret": conf.Exchange.ExchangeSecret,
	})

	return &Monitor{
		Exchange: exchange,
	}, nil
}
