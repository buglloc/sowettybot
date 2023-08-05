package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/buglloc/sowettybot/internal/config"
)

func notificationInfo(n Notification) string {
	return fmt.Sprintf(
		"threshold=%.4f last_rate=%.4f last_send=%2.fh",
		n.Threshold, n.lastRate, time.Since(n.lastSend).Hours(),
	)
}

func TestShouldNotify(t *testing.T) {
	type ratesCase struct {
		rate     float64
		expected bool
	}

	cases := []struct {
		notification Notification
		rates        []ratesCase
	}{
		{
			notification: Notification{
				Notification: config.Notification{
					Threshold: 3.0,
				},
			},
			rates: []ratesCase{
				{
					rate:     3.0,
					expected: true,
				},
				{
					rate:     2.9,
					expected: true,
				},
				{
					rate:     2.8,
					expected: true,
				},
				{
					rate:     0.0,
					expected: false,
				},

				{
					rate:     3.1,
					expected: false,
				},
			},
		},
		{
			notification: Notification{
				Notification: config.Notification{
					Threshold: 3.0,
				},
				lastRate: 2.9,
			},
			rates: []ratesCase{
				{
					rate:     3.0,
					expected: true,
				},
				{
					rate:     2.9,
					expected: true,
				},
				{
					rate:     2.8,
					expected: true,
				},
				{
					rate:     0.0,
					expected: false,
				},

				{
					rate:     3.1,
					expected: false,
				},
			},
		},
		{
			notification: Notification{
				Notification: config.Notification{
					Threshold: 3.0,
				},
				lastRate: 2.9,
				lastSend: time.Now().Add(-12 * time.Hour),
			},
			rates: []ratesCase{
				{
					rate:     3.0,
					expected: true,
				},
				{
					rate:     2.9,
					expected: true,
				},
				{
					rate:     2.8,
					expected: true,
				},
				{
					rate:     0.0,
					expected: false,
				},

				{
					rate:     3.1,
					expected: false,
				},
			},
		},
		{
			notification: Notification{
				Notification: config.Notification{
					Threshold: 3.0,
				},
				lastRate: 2.9,
				lastSend: time.Now(),
			},
			rates: []ratesCase{
				{
					rate:     3.0,
					expected: false,
				},
				{
					rate:     2.902,
					expected: false,
				},
				{
					rate:     2.901,
					expected: false,
				},
				{
					rate:     2.9,
					expected: false,
				},
				{
					rate:     2.899,
					expected: false,
				},
				{
					rate:     2.898,
					expected: true,
				},
				{
					rate:     2.8,
					expected: true,
				},
				{
					rate:     0.0,
					expected: false,
				},

				{
					rate:     3.1,
					expected: false,
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(notificationInfo(tc.notification), func(t *testing.T) {
			for _, rc := range tc.rates {
				t.Run(fmt.Sprint(rc.expected), func(t *testing.T) {
					require.Equal(t, rc.expected, tc.notification.ShouldNotify(rc.rate))
				})
			}
		})
	}
}
