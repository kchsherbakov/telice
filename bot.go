package main

import (
	"fmt"
	tbot "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
)

type bot struct {
	token          string
	yandexClientId string
	api            *tbot.BotAPI
}

func NewBot(token string, yandexClientId string) *bot {
	api, err := tbot.NewBotAPI(token)
	if err != nil {
		log.Fatalln("Could not create a new bot API instance", err)
	}

	log.Printf("Bot has started. Authorized on account %s", api.Self.UserName)

	return &bot{token, yandexClientId, api}
}

func (b *bot) Run() {
	u := tbot.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			b.handleCommand(update.Message)
			continue
		}

		log.Print(update.Message.Text)
	}
}

func (b *bot) handleCommand(msg *tbot.Message) {
	cmd := msg.Command()
	log.Println(msg.CommandArguments())
	if cmd == StartCmd {
		text := `
Hello! To use telice first we need to authenticate you. Please, click on the link down below to authenticate. 
Authentication is done thought Yandex.OAuth. I will never ask you for login or password.
`
		out := tbot.NewMessage(msg.Chat.ID, text)
		b.api.Send(out)

		link := fmt.Sprintf("https://oauth.yandex.com/authorize?response_type=token&client_id=%v", b.yandexClientId)
		out = tbot.NewMessage(msg.Chat.ID, link)
		b.api.Send(out)
	}
}
