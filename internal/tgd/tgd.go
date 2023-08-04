package tgd

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

type Server struct {
	bot        *telego.Bot
	rtc        *rateit.Client
	history    *history.History
	renderer   *renderer.HistoryRenderer
	exchanges  []config.Exchange
	limits     config.Limits
	ratesCache *ttlcache.Cache[string, models.Rate]
	closed     chan struct{}
	ctx        context.Context
	cancelCtx  context.CancelFunc
}

func NewServer(cfg *config.Config) (*Server, error) {
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

	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		bot:       bot,
		rtc:       rtc,
		history:   history.NewHistory(cfg.History.StorageFile, cfg.Limits.History.Overall),
		renderer:  renderer.NewHistoryRenderer(),
		exchanges: cfg.Exchanges,
		limits:    cfg.Limits,
		ratesCache: ttlcache.New[string, models.Rate](
			ttlcache.WithTTL[string, models.Rate](5 * time.Minute),
		),
		closed:    make(chan struct{}),
		ctx:       ctx,
		cancelCtx: cancel,
	}, nil
}

func (s *Server) Start() error {
	if err := s.bot.Run(); err != nil {
		return fmt.Errorf("run failed: %w", err)
	}

	defer close(s.closed)

	if err := s.initHandlers(); err != nil {
		return fmt.Errorf("unable to register handlers: %w", err)
	}

	cacheTicker := time.NewTicker(30 * time.Minute)
	defer cacheTicker.Stop()

	//Monitores any other update
	updateChannel := *s.bot.GetUpdateChannel()
	for {
		select {
		case <-cacheTicker.C:
			s.ratesCache.DeleteExpired()
		case u := <-updateChannel:
			_, _ = s.bot.SendMessage(u.Message.Chat.Id, "Sorry, unsupported command", "", u.Message.MessageId, false, false)
		case <-s.ctx.Done():
			return nil
		}
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.cancelCtx()

	select {
	case <-ctx.Done():
		return errors.New("shutdown time out")
	case <-s.closed:
		return nil
	}
}
