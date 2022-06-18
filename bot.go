package main

import (
	"encoding/base64"
	"fmt"
	tbot "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"net/http"
	"os"
)

var httpClient = &http.Client{}

type bot struct {
	token           string
	api             *tbot.BotAPI
	sessionProvider SessionProvider
}

func NewBot(token string) *bot {
	api, err := tbot.NewBotAPI(token)
	if err != nil {
		log.Fatalln("Could not create a new bot API instance", err)
	}

	log.Printf("Bot has started. Authorized on account %s", api.Self.UserName)

	sp := NewInMemorySessionProvider()
	return &bot{token, api, sp}
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

func (b *bot) send(chatId int64, text string) {
	msg := tbot.NewMessage(chatId, text)
	_, err := b.api.Send(msg)
	if err != nil {
		log.Println(fmt.Sprintf("Error occurred while trying to send the message to chat %v", chatId), err)
	}
}

func (b *bot) handleCommand(msg *tbot.Message) {
	cmd := msg.Command()
	args := msg.CommandArguments()

	switch cmd {
	case StartCmd:
		b.handleStartCommand(msg.Chat.ID, args)
	}
}

func (b *bot) handleStartCommand(chatId int64, args string) {
	if s, ok := b.sessionProvider.TryGet(chatId); ok {
		text := "Looks like everything is ready. Feel free to send me a link to share with your Alice."
		b.send(s.chatId, text)
		return
	}

	if args != "" {
		decoded, err := base64.StdEncoding.DecodeString(args)
		if err != nil {
			log.Println(fmt.Sprintf("Error occurred decoding base64 string `%v`", decoded), err)
			goto hello
		}

		yaClient := NewYandexClient(httpClient)
		err = yaClient.SetupTokens(string(decoded))
		if err != nil {
			b.send(chatId, "Could not complete authentication process. Please, try again.")
		}

		s := NewSession(chatId, yaClient)
		b.sessionProvider.SaveOrUpdate(s)

		b.send(chatId, "Authentication is complete.\nSend me a link and I will share it with Alice. Have fun!")

		return
	}

hello:
	text := `
Hello!
To use telice first we need to authenticate you. Please, click on the link down below to authenticate. 
Authentication is done thought Yandex.OAuth. I will never ask you for login or password.
	`
	b.send(chatId, text)

	link := fmt.Sprintf("https://oauth.yandex.com/authorize?response_type=token&client_id=%v", os.Getenv(YandexClientId))
	b.send(chatId, link)

	return
}
