package margin_monitor

import (
	"context"
	"log"
	"margin_monitor/config"
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
				if *ps.MarginRatio > c.Conf.Monitor.DangerThreshold {
					go c.M.AddMargin(*ps.Symbol, *ps.InitialMargin/2)
				}
			}
		}
	}
}
