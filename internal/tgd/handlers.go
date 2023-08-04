package tgd

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
	"github.com/buglloc/sowettybot/internal/models"
	"github.com/buglloc/sowettybot/internal/renderer"
)

const (
	tgMdMode = "MarkdownV2"
)

func (s *Server) initHandlers() error {
	toRegister := map[string]func(u *objects.Update){
		"/start":       s.handleStart,
		"/rates":       s.handleRates,
		"/history":     s.handleHistoryChart,
		"/longhistory": s.handleLongHistoryChart,
		"/rawhistory":  s.handleHistoryText,
	}

	for pattern, handler := range toRegister {
		if err := s.bot.AddHandler(pattern, handler, "private"); err != nil {
			return fmt.Errorf("unable to register handler %q: %w", pattern, err)
		}
	}

	return nil
}

func (s *Server) handleStart(u *objects.Update) {
	err := s.sendMdMessage(
		u.Message.Chat.Id,
		"Nice to see you, type /history to get exchange rates history ;)",
		u.Message.MessageId,
	)
	if err != nil {
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("unable to send reply")
	}
}

func (s *Server) handleRates(u *objects.Update) {
	_, _ = s.bot.SendMessage(u.Message.Chat.Id, "I'll check exchange rates...please be patient...", "", u.Message.MessageId, true, false)

	reply, err := s.renderRates()
	if err != nil {
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("failed to generate rates")
		reply = fmt.Sprintf("shit happens: %v", err)
	}

	err = s.sendMdMessage(u.Message.Chat.Id, reply, 0)
	if err != nil {
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("unable to send reply")
	}
}

func (s *Server) handleHistoryText(u *objects.Update) {
	reply, err := func() (string, error) {
		entries, err := s.history.Entries(24)
		if err != nil {
			return "", fmt.Errorf("get entries: %w", err)
		}

		if len(entries) == 0 {
			return "Sorry, history is unavailable so far", nil
		}

		return s.renderer.Log(entries)
	}()

	if err != nil {
		reply = fmt.Sprintf("ooops, shit happens: %v", err)
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("failed to get history")
	}

	err = s.sendMdMessage(u.Message.Chat.Id, reply, u.Message.MessageId)
	if err != nil {
		log.Error().Err(err).Int("chat_id", u.Message.Chat.Id).Msg("unable to send reply")
	}
}

func (s *Server) handleHistoryChart(u *objects.Update) {
	s.sendHistoryChart(u, s.limits.History.Short)
}

func (s *Server) handleLongHistoryChart(u *objects.Update) {
	s.sendHistoryChart(u, s.limits.History.Long)
}

func (s *Server) sendHistoryChart(u *objects.Update, limit int) {
	sendHistory := func() error {
		entries, err := s.history.Entries(limit)
		if err != nil {
			return fmt.Errorf("get entries: %w", err)
		}

		if len(entries) == 0 {
			_ = s.sendMdMessage(
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
		startDate, endDate, err := s.renderer.Graph(entries, graphF, cfg)
		if err != nil {
			return err
		}

		ms := s.bot.SendPhoto(
			u.Message.Chat.Id,
			u.Message.MessageId,
			fmt.Sprintf(
				"`%s -> %s`",
				startDate.Format("02 Jan 15:04"),
				endDate.Format("02 Jan 15:04"),
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
		_ = s.sendMdMessage(
			u.Message.Chat.Id,
			fmt.Sprintf("ooops, shit happens: %v", err),
			u.Message.MessageId,
		)
	}
}

func (s *Server) sendMdMessage(chatID int, text string, replyTo int) error {
	_, err := s.bot.SendMessage(chatID, renderer.EscapeTgMd(text), tgMdMode, replyTo, false, false)
	return err
}

func (s *Server) renderRates() (string, error) {
	var wg sync.WaitGroup
	wg.Add(len(s.exchanges))
	rates := make(models.Rates, len(s.exchanges))
	for i, ex := range s.exchanges {
		go func(i int, ex config.Exchange) {
			defer wg.Done()

			cached := s.ratesCache.Get(ex.Route)
			if cached != nil && !cached.IsExpired() {
				rates[i] = cached.Value()
				return
			}

			rate, err := s.rtc.Rate(context.Background(), ex.Route)
			if err != nil {
				log.Error().Err(err).Str("route", ex.Route).Msg("unable to fetch rates")
			}

			rate.Name = ex.Name

			s.ratesCache.Set(ex.Route, rate, ttlcache.DefaultTTL)
			rates[i] = rate
		}(i, ex)
	}
	wg.Wait()

	reply, err := s.renderer.Rates(rates)
	if err != nil {
		return fmt.Sprintf("Sotty, shit happens: %v", err), nil
	}

	return reply, nil
}
