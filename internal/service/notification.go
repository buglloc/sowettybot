package service

import (
	"math"
	"time"

	"github.com/buglloc/sowettybot/internal/config"
)

const (
	notifyThreshold       = 60 * time.Minute
	rateEqualityThreshold = 0.001
)

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
	if compareRate(rate, 0.0) == 0 {
		return false
	}

	if compareRate(rate, n.Threshold) == 1 {
		return false
	}

	if compareRate(rate, n.lastRate) == -1 {
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

func compareRate(a, b float64) int {
	switch {
	case math.Abs(a-b) <= rateEqualityThreshold:
		return 0
	case a < b:
		return -1
	case a > b:
		return 1
	}

	return 0
}
