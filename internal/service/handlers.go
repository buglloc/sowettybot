package service

import (
	"context"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/SakoDroid/telego/objects"
	"github.com/jellydator/ttlcache/v3"
	"github.com/rs/zerolog/log"

	"github.com/buglloc/sowettybot/internal/config"
	"github.com/buglloc/sowettybot/internal/history"
	"github.com/buglloc/sowettybot/internal/models"
	"github.com/buglloc/sowettybot/internal/rateit"
	"github.com/buglloc/sowettybot/internal/renderer"
)

type CommandsHandler struct {
	bot        *BotWrapper
	rtc        *rateit.Client
	history    *history.History
	renderer   *renderer.HistoryRenderer
	exchanges  []config.Exchange
	limits     config.Limits
	ratesCache *ttlcache.Cache[string, models.Rate]
}

func (h *CommandsHandler) Initialize() error {
	toRegister := map[string]func(u *objects.Update){
		"/start":       h.handleStart,
		"/chatid":      h.handleChatID,
		"/rates":       h.handleRates,
		"/history":     h.handleHistoryChart,
		"/longhistory": h.handleLongHistoryChart,
		"/rawhistory":  h.handleHistoryText,
	}

	for pattern, handler := range toRegister {
		if err := h.bot.AddHandler(pattern, handler, "private"); err != nil {
			return fmt.Errorf("unable to register handler %q: %w", pattern, err)
		}
	}

	return nil
}

func (h *CommandsHandler) Tick() {
	h.ratesCache.DeleteExpired()
}

func (h *CommandsHandler) handleStart(u *objects.Update) {
	err := h.bot.SendMdMessage(
		u.Message.Chat.Id,
		"Nice to see you, type /history to get exchange rates history ;)",
		u.Message.MessageId,
	)
	if err != nil {
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("unable to send reply")
	}
}

func (h *CommandsHandler) handleChatID(u *objects.Update) {
	err := h.bot.SendMdMessage(
		u.Message.Chat.Id,
		fmt.Sprintf("chat id: `%d`", u.Message.Chat.Id),
		u.Message.MessageId,
	)
	if err != nil {
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("unable to send reply")
	}
}

func (h *CommandsHandler) handleRates(u *objects.Update) {
	_, _ = h.bot.SendMessage(u.Message.Chat.Id, "I'll check exchange rates...please be patient...", "", u.Message.MessageId, true, false)

	reply, err := h.renderRates()
	if err != nil {
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("failed to generate rates")
		reply = fmt.Sprintf("shit happens: %v", err)
	}

	err = h.bot.SendMdMessage(u.Message.Chat.Id, reply, 0)
	if err != nil {
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("unable to send reply")
	}
}

func (h *CommandsHandler) handleHistoryText(u *objects.Update) {
	reply, err := func() (string, error) {
		entries, err := h.history.Entries(24)
		if err != nil {
			return "", fmt.Errorf("get entries: %w", err)
		}

		if len(entries) == 0 {
			return "Sorry, history is unavailable so far", nil
		}

		return h.renderer.Log(entries)
	}()

	if err != nil {
		reply = fmt.Sprintf("ooops, shit happens: %v", err)
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("failed to get history")
	}

	err = h.bot.SendMdMessage(u.Message.Chat.Id, reply, u.Message.MessageId)
	if err != nil {
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("unable to send reply")
	}
}

func (h *CommandsHandler) handleHistoryChart(u *objects.Update) {
	h.sendHistoryChart(u, h.limits.History.Short)
}

func (h *CommandsHandler) handleLongHistoryChart(u *objects.Update) {
	h.sendHistoryChart(u, h.limits.History.Long)
}

func (h *CommandsHandler) sendHistoryChart(u *objects.Update, limit int) {
	sendHistory := func() error {
		entries, err := h.history.Entries(limit)
		if err != nil {
			return fmt.Errorf("get entries: %w", err)
		}

		if len(entries) == 0 {
			_ = h.bot.SendMdMessage(
				u.Message.Chat.Id,
				"Sorry, history is unavailable so far",
				u.Message.MessageId,
			)
			return nil
		}

		graphF, err := os.CreateTemp("", "sowetty-history-*.png")
		if err != nil {
			return fmt.Errorf("create temporary file: %w", err)
		}
		defer func() {
			_ = graphF.Close()
			_ = os.RemoveAll(graphF.Name())
		}()

		width := math.Ceil(float64(len(entries))/192) * 512
		cfg := renderer.NewGraphConfig().
			Width(int(width)).
			Height(512).
			WithSMA(true)
		startDate, endDate, err := h.renderer.Graph(entries, graphF, cfg)
		if err != nil {
			return err
		}

		ms := h.bot.SendPhoto(
			u.Message.Chat.Id,
			u.Message.MessageId,
			fmt.Sprintf(
				"`%s -> %s`",
				startDate.Format("02 Jan 15:04 MST"),
				endDate.Format("02 Jan 15:04 MST"),
			),
			tgMdMode,
		)

		graphF, err = os.Open(graphF.Name())
		if err != nil {
			return fmt.Errorf("open temporary file: %w", err)
		}
		defer func() { _ = graphF.Close() }()

		_, err = ms.SendByFile(graphF, false, false)
		return err
	}

	if err := sendHistory(); err != nil {
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("failed to send history")
		_ = h.bot.SendMdMessage(
			u.Message.Chat.Id,
			fmt.Sprintf("ooops, shit happens: %v", err),
			u.Message.MessageId,
		)
	}
}

func (h *CommandsHandler) renderRates() (string, error) {
	var wg sync.WaitGroup
	wg.Add(len(h.exchanges))
	rates := make(models.Rates, len(h.exchanges))
	for i, ex := range h.exchanges {
		go func(i int, ex config.Exchange) {
			defer wg.Done()

			cached := h.ratesCache.Get(ex.Route)
			if cached != nil && !cached.IsExpired() {
				rates[i] = cached.Value()
				return
			}

			rate, err := h.rtc.Rate(context.Background(), ex.Route)
			if err != nil {
				log.Error().Err(err).Str("route", ex.Route).Msg("unable to fetch rates")
			}

			rate.Name = ex.Name

			h.ratesCache.Set(ex.Route, rate, ttlcache.DefaultTTL)
			rates[i] = rate
		}(i, ex)
	}
	wg.Wait()

	reply, err := h.renderer.Rates(rates)
	if err != nil {
		return fmt.Sprintf("Sotty, shit happens: %v", err), nil
	}

	return reply, nil
}
