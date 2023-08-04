package service

import (
	"github.com/SakoDroid/telego"

	"github.com/buglloc/sowettybot/internal/renderer"
)

const (
	tgMdMode = "MarkdownV2"
)

type BotWrapper struct {
	*telego.Bot
}

func (b *BotWrapper) SendMdMessage(chatID int, text string, replyTo int) error {
	_, err := b.Bot.SendMessage(chatID, renderer.EscapeTgMd(text), tgMdMode, replyTo, false, false)
	return err
}
