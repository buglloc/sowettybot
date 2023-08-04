package service

import (
	"time"

	"github.com/buglloc/sowettybot/internal/config"
)

const notifyThreshold = 60 * time.Minute

type Notification struct {
	config.Notification
	lastRate float64
	lastSend time.Time
}

func NewNotification(cfg config.Notification) *Notification {
	return &Notification{
		Notification: cfg,
	}
}

func (n *Notification) ShouldNotify(rate float64) bool {
	if rate > n.Threshold {
		return false
	}

	if rate < n.lastRate {
		return true
	}

	if time.Since(n.lastSend) > notifyThreshold {
		return true
	}

	return false
}

func (n *Notification) Notified(rate float64) {
	n.lastRate = rate
	n.lastSend = time.Now()
}
