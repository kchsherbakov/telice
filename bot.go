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
			err := b.handleCommand(update.Message)
			if err != nil {
				b.handleError(update.FromChat().ID, err)
				continue
			}
		}
	}
}

func (b *bot) handleError(chatId int64, err error) {
	if err == nil {
		return
	}

	if e, ok := err.(*botError); ok {
		b.send(chatId, e.Error())
		return
	}

	b.send(chatId, "Something went wrong. Please, try again.")
}

func (b *bot) send(chatId int64, text string) {
	msg := tbot.NewMessage(chatId, text)
	_, err := b.api.Send(msg)
	if err != nil {
		log.Println(fmt.Sprintf("Error occurred while trying to send the message to chat %v", chatId), err)
	}
}

func (b *bot) isAuthorizationRequired(cmd string) bool {
	if cmd == StartCmd {
		return false
	}

	return true
}

func (b *bot) handleCommand(msg *tbot.Message) error {
	cmd := msg.Command()
	args := msg.CommandArguments()

	s, found := b.sessionProvider.TryGet(msg.Chat.ID)
	if !found && b.isAuthorizationRequired(cmd) {
		return NewBotError(fmt.Sprintf("Authentication required. Please, click /%s to initiate.", StartCmd))
	}

	switch cmd {
	case StartCmd:
		return b.handleStartCommand(msg.Chat.ID, args)
	case ListDevicesCmd:
		return b.handleListDevicesCommand(s)
	case SelectAsDefaultCmd:
		return b.handleSelectAsDefaultCommand(s)
	}

	return nil
}

func (b *bot) handleStartCommand(chatId int64, args string) error {
	if s, ok := b.sessionProvider.TryGet(chatId); ok {
		text := "Looks like everything is ready. Feel free to send me a link to share with your Alice."
		b.send(s.chatId, text)
		return nil
	}

	if args != "" {
		decoded, err := base64.StdEncoding.DecodeString(args)
		if err != nil {
			log.Println(fmt.Sprintf("Error occurred decoding base64 string `%v`", decoded), err)
			goto hello
		}

		oauthToken, csrfToken, err := b.yaClient.getTokens(string(decoded))
		if err != nil {
			return NewBotError("Could not complete authentication process. Please, try again.")
		}

		s := NewSession(chatId, oauthToken, csrfToken)
		b.sessionProvider.SaveOrUpdate(s)

		b.send(chatId, "Authentication is complete.\nSend me a link and I will share it with Alice. Have fun!")

		return nil
	}

hello:
	text := `
Hello!
To use telice first we need to authenticate you. Please, click on the link down below to authenticate. 
Authentication is done thought Yandex.OAuth. I will never ask you for login or password.
	`
	b.send(chatId, text)
	b.send(chatId, b.yaClient.getOAuthUrl())

	return nil
}

func (b *bot) handleListDevicesCommand(s *session) error {
	devices, err := b.yaClient.getYandexStations(s)
	if err != nil {
		return NewBotError("Could not get list of registered devices. Please, try again.")
	}

	if len(devices) == 0 {
		return NewBotError("I didn't find any yandex stations. Are they configured properly?")
	}

	iotInfo, _ := b.yaClient.getYandexSmartHomeInfo(s)

	msgText := b.formatYandexStationsMessage(devices, iotInfo.Rooms, iotInfo.Households)
	b.send(s.chatId, msgText)

	return nil
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

func (b *bot) handleSelectAsDefaultCommand(s *session) error {
	return nil
}
