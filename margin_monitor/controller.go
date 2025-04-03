package margin_monitor

import (
	"context"
	"log"
	"margin_monitor/config"
	"math"
	"time"
)

func NewController(conf *config.Config) (*Controller, error) {
	m, err := NewMonitor(conf)
	if err != nil {
		log.Fatalf("init monitor err: %v", err)
	}
	return &Controller{
		Conf: conf,
		M:    m,
	}, nil
}

type Controller struct {
	Conf *config.Config
	M    *Monitor
}

func (c *Controller) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second * time.Duration(c.Conf.Monitor.CheckInterval)):
			positions, err := c.M.FetchPositions()
			if err != nil {
				log.Printf("fetch positions error: %v\n", err)
				continue
			}
			for i := range positions {
				ps := positions[i]
				log.Printf("Checking position: Symbol=%s, MarginRatio=%.4f, InitialMargin=%.4f\n",
					*ps.Symbol, *ps.MarginRatio, *ps.InitialMargin)
				if *ps.MarginRatio > c.Conf.Monitor.DangerThreshold {
					log.Printf("⚠️ Margin ratio exceeds threshold! Adding margin: Symbol=%s, Amount=%.4f\n",
						*ps.Symbol, *ps.InitialMargin/2)
					// 计算持仓量
					amount := *ps.InitialMargin
					if *ps.InitialMargin > 20 {
						amount = *ps.InitialMargin / 2
					}
					amount = math.Ceil(amount)
					go c.M.AddMargin(*ps.Symbol, amount)
				}
			}
		}
	}
}
