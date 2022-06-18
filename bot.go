package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	tbot "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"net/http"
)

type bot struct {
	token           string
	api             *tbot.BotAPI
	yaClient        *YandexClient
	sessionProvider SessionProvider
	cacheProvider   CacheProvider
}

func NewBot(token string, yandexClientId string) *bot {
	api, err := tbot.NewBotAPI(token)
	if err != nil {
		log.Fatalln("Could not create a new bot API instance", err)
	}

	log.Printf("Bot has started. Authorized on account %s", api.Self.UserName)

	sp := NewInMemorySessionProvider()
	cp := NewInMemoryCacheProvider()
	yc := NewYandexClient(yandexClientId, cp, &http.Client{})

	return &bot{token, api, yc, sp, cp}
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
	case ListDevices:
		b.handleListDevicesCommand(msg.Chat.ID)
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

		oauthToken, csrfToken, err := b.yaClient.getTokens(string(decoded))
		if err != nil {
			b.send(chatId, "Could not complete authentication process. Please, try again.")
		}

		s := NewSession(chatId, oauthToken, csrfToken)
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
	b.send(chatId, b.yaClient.getOAuthUrl())

	return
}

func (b *bot) handleListDevicesCommand(chatId int64) {
	s, ok := b.sessionProvider.TryGet(chatId)
	if !ok {
		text := "Please authenticate to perform this action"
		b.send(chatId, text)
		return
	}

	devices, err := b.yaClient.getYandexStations(s)
	if err != nil {
		b.send(s.chatId, "Could not get list of registered devices. Please, try again")
		return
	}

	if len(devices) == 0 {
		b.send(s.chatId, "I didn't find any yandex stations. Is it configured properly?")
		return
	}

	iotInfo, _ := b.yaClient.getYandexSmartHomeInfo(s)

	msgText := b.formatYandexStationsMessage(devices, iotInfo.Rooms, iotInfo.Households)
	b.send(s.chatId, msgText)
}

func (b *bot) formatYandexStationsMessage(devices []device, rooms []room, households []household) string {
	var buf = bytes.Buffer{}
	for i, d := range devices {
		var r room
		for _, v := range rooms {
			if v.Id == d.Room {
				r = v
				break
			}
		}

		var h household
		for _, v := range households {
			if v.Id == r.HouseholdId {
				h = v
				break
			}
		}

		buf.WriteString(fmt.Sprintf("%d. %s - %s - %s\n", i+1, h.Name, r.Name, d.Name))
	}

	return buf.String()
}
