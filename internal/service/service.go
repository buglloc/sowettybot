package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/SakoDroid/telego"
	"github.com/SakoDroid/telego/configs"
	"github.com/jellydator/ttlcache/v3"

	"github.com/buglloc/sowettybot/internal/config"
	"github.com/buglloc/sowettybot/internal/history"
	"github.com/buglloc/sowettybot/internal/models"
	"github.com/buglloc/sowettybot/internal/rateit"
	"github.com/buglloc/sowettybot/internal/renderer"
)

type Service struct {
	handlers  *CommandsHandler
	notifier  *Notifier
	bot       *BotWrapper
	closed    chan struct{}
	ctx       context.Context
	cancelCtx context.CancelFunc
}

func NewService(cfg *config.Config) (*Service, error) {
	rtc, err := rateit.NewClient(
		rateit.WithUpstream(cfg.RateIT.Upstream),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create rateit client: %w", err)
	}

	up := configs.DefaultUpdateConfigs()
	botCfg := configs.BotConfigs{
		BotAPI:         configs.DefaultBotAPI,
		APIKey:         cfg.Telegram.APIKey,
		UpdateConfigs:  up,
		Webhook:        false,
		LogFileAddress: configs.DefaultLogFile,
	}

	bot, err := telego.NewBot(&botCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to create bot: %w", err)
	}

	notifications := make([]*Notification, len(cfg.Notifier.Notifications))
	for i, n := range cfg.Notifier.Notifications {
		notifications[i] = NewNotification(n)
	}

	bw := &BotWrapper{Bot: bot}
	hist := history.NewHistory(cfg.History.StorageFile, cfg.Limits.History.Overall)

	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		handlers: &CommandsHandler{
			bot:       bw,
			rtc:       rtc,
			history:   hist,
			renderer:  renderer.NewHistoryRenderer(),
			exchanges: cfg.Exchanges,
			limits:    cfg.Limits,
			ratesCache: ttlcache.New[string, models.Rate](
				ttlcache.WithTTL[string, models.Rate](5 * time.Minute),
			),
		},
		notifier: &Notifier{
			bot:           bw,
			history:       hist,
			notifications: notifications,
			checkPeriod:   cfg.Notifier.CheckPeriod,
		},
		bot:       bw,
		closed:    make(chan struct{}),
		ctx:       ctx,
		cancelCtx: cancel,
	}, nil
}

func (s *Service) Start() error {
	if err := s.bot.Run(); err != nil {
		return fmt.Errorf("run failed: %w", err)
	}

	defer close(s.closed)

	if err := s.notifier.Initialize(); err != nil {
		return fmt.Errorf("unable to register handlers: %w", err)
	}

	if err := s.handlers.Initialize(); err != nil {
		return fmt.Errorf("unable to register handlers: %w", err)
	}

	updateTicker := time.NewTicker(1 * time.Minute)
	defer updateTicker.Stop()

	//Monitores any other update
	updateChannel := *s.bot.GetUpdateChannel()
	for {
		select {
		case <-updateTicker.C:
			s.handlers.Tick()
			s.notifier.Tick()
		case u := <-updateChannel:
			_, _ = s.bot.SendMessage(u.Message.Chat.Id, "Sorry, unsupported command", "", u.Message.MessageId, false, false)
		case <-s.ctx.Done():
			return nil
		}
	}
}

func (s *Service) Shutdown(ctx context.Context) error {
	s.cancelCtx()

	select {
	case <-ctx.Done():
		return errors.New("shutdown time out")
	case <-s.closed:
		return nil
	}
}
