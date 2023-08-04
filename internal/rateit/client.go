package rateit

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/buglloc/sowettybot/internal/models"
)

const (
	DefaultUpstream = "http://localhost:3000"
	DefaultRetries  = 3
	DefaultTimeout  = 5 * time.Minute
)

type Client struct {
	httpc *resty.Client
	log   zerolog.Logger
}

func NewClient(opts ...Option) (*Client, error) {
	client := &Client{
		log: log.With().Str("source", "koronapay").Logger(),
		httpc: resty.New().
			SetRetryCount(DefaultRetries).
			SetTimeout(DefaultTimeout),
	}

	defaultOpts := []Option{
		WithUpstream(DefaultUpstream),
	}

	for _, opt := range defaultOpts {
		opt(client)
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

func (c *Client) Rate(ctx context.Context, route string) (models.Rate, error) {
	c.log.Info().
		Any("route", route).
		Msg("fetch rate")

	var rsp RateRsp
	var remoteErr ErrorRsp
	httpRsp, err := c.httpc.R().
		SetContext(ctx).
		SetError(&remoteErr).
		SetResult(&rsp).
		Get("/api/v1/rate/" + route)

	out := models.Rate{
		When: time.Now(),
	}
	if err != nil {
		return out, fmt.Errorf("request failed: %w", err)
	}

	if remoteErr.Code != 0 {
		return out, fmt.Errorf("remote error %q: %s", remoteErr.Code, remoteErr.Message)
	}

	if !httpRsp.IsSuccess() {
		return out, fmt.Errorf("non-200 status code: %s", httpRsp.Status())
	}

	out.Rate = rsp.Rate
	return out, nil
}
