package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/buglloc/sowettybot/internal/config"
	"github.com/buglloc/sowettybot/internal/history"
)

type Notifier struct {
	bot           *BotWrapper
	history       *history.History
	notifications []config.Notification
	checkPeriod   time.Duration
	lastTick      time.Time
	lastCheck     time.Time
}

func (n *Notifier) Initialize() error {
	return nil
}

func (n *Notifier) Tick() {
	now := time.Now()
	if now.Sub(n.lastTick) < n.checkPeriod {
		return
	}

	n.lastTick = now
	entries, err := n.history.Entries(1)
	if err != nil {
		log.Error().Err(err).Msg("unable to get history")
		return
	}

	if len(entries) == 0 {
		return
	}

	entry := entries[0]
	if n.lastCheck.After(entry.When) {
		return
	}

	n.lastCheck = now
	var notification strings.Builder
	for _, cfg := range n.notifications {
		notification.Reset()
		for i, v := range entry.Values {
			if v > cfg.Threshold {
				continue
			}

			if notification.Len() == 0 {
				_, _ = fmt.Fprintf(&notification, "YAY! Nice exchange rate (threshold is %d)!\n", cfg.Threshold)
			}
			_, _ = fmt.Fprintf(&notification, "%s: %.2f\n", entry.Names[i], v)
		}

		if notification.Len() > 0 {
			msg := notification.String()
			err := n.bot.SendMdMessage(cfg.ChatID, msg, 0)
			if err != nil {
				log.Error().Err(err).Str("message", msg).Msg("unable to send notification")
			}
		}
	}
}
