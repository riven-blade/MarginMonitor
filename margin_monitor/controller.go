package margin_monitor

import (
	"context"
	"margin_monitor/config"
	"time"
)

func NewController(conf *config.Config) (*Controller, error) {
	m, err := NewMonitor(conf)
	if err != nil {
		return nil, err
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
		default:
			time.Sleep(time.Second * time.Duration(c.Conf.Monitor.CheckInterval))
			// 触发检查动作

		}
	}
}
