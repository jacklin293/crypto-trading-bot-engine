package message

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Telegram struct {
	bot *tgbotapi.BotAPI
}

func newTelegram(token string) (m Messenger, err error) {
	var bot *tgbotapi.BotAPI
	bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return
	}
	tg := &Telegram{
		bot: bot,
	}
	return tg, err
}

func (tg *Telegram) Send(chatId int64, text string) {
	var msg tgbotapi.MessageConfig
	msg = tgbotapi.NewMessage(chatId, text)
	tg.bot.Send(msg)
}
