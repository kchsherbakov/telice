package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	tbot "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"net/http"
	"regexp"
	"sort"
	"strings"
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
		log.WithError(err).Fatal("Could not create a new bot API instance")
	}

	log.Infof("Bot has started. Authorized on account %s", api.Self.UserName)

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
		s, found := b.sessionProvider.TryGet(update.FromChat().ID)
		if !found && b.isAuthorizationRequired(&update) {
			b.handleError(update.FromChat().ID, NewBotError(fmt.Sprintf("Authentication required. Please, click /%s to initiate.", StartCmd)))
			continue
		}

		if update.Message != nil {
			handled, err := b.tryHandleCommandMessage(s, update)
			if !handled {
				err = b.handleMessage(s, update.Message)
			}

			b.handleError(update.FromChat().ID, err)
		} else if update.CallbackQuery != nil {
			b.handleCallbackQuery(s, update.CallbackQuery)
		}
	}
	log.Infof("FINISHED")
}

func (b *bot) handleCallbackQuery(s *session, callback *tbot.CallbackQuery) {
	if callback.Data == "" {
		return
	}

	parts := strings.Split(callback.Data, ":")
	method, data := parts[0], parts[1]

	switch method {
	case SelectAsDefaultCallback:
		b.handleSelectAsDefaultCommandCallback(s, data)
	case OneTimePlayMediaCallback:
		b.handleOneTimePlayMediaCallback(s, callback.Message.ReplyToMessage, data)
	}
}

func (b *bot) handleMessage(s *session, msg *tbot.Message) error {
	r, _ := regexp.Compile(URLRegexPattern)
	match := r.Find([]byte(msg.Text))
	if match == nil {
		return NewBotError("URL link is not found in the message. Please, send me a valid one.")
	}
	url := string(match)

	// region YouTube

	loweredUrl := strings.ToLower(url)
	if !strings.Contains(loweredUrl, "youtube") && !strings.Contains(loweredUrl, "youtu.be") {
		return NewBotError("Sorry, but I support only YouTube at the moment :(")
	}
	url = reformatYouTubeUrl(url)

	// endregion

	devices, err := b.yaClient.getYandexStations(s)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return NewBotError("I didn't find any yandex stations. Are they configured properly?")
	}

	if s.defaultDevice != nil {
		for _, d := range devices {
			if d.Id == s.defaultDevice.Id {
				return b.yaClient.playMedia(s, nil, url)
			}
		}

		return NewBotError("Selected device is not currently available. Please, try again later.")
	}

	if len(devices) == 1 {
		return b.yaClient.playMedia(s, &devices[0], url)
	}

	iotInfo, _ := b.yaClient.getYandexSmartHomeInfo(s)

	strs := b.yandexStationsToString(s, devices, iotInfo.Rooms, iotInfo.Households)

	rows := make([][]tbot.InlineKeyboardButton, 0)
	for i, s := range strs {
		// Notice, Telegram api requires button data to be 64 bytes or less
		data := fmt.Sprintf("%s:%s", OneTimePlayMediaCallback, devices[i].Id)
		rows = append(rows, tbot.NewInlineKeyboardRow(tbot.NewInlineKeyboardButtonData(s, data)))
	}
	keyboard := tbot.NewInlineKeyboardMarkup(rows...)

	replyMsg := tbot.NewMessage(s.chatId, "Please, select the station you want to share media with.")
	replyMsg.ReplyToMessageID = msg.MessageID
	replyMsg.ReplyMarkup = keyboard

	//goland:noinspection GoUnhandledErrorResult
	b.api.Send(replyMsg)

	return nil
}

func (b *bot) tryHandleCommandMessage(s *session, update tbot.Update) (bool, error) {
	if !update.Message.IsCommand() {
		return false, nil
	}

	return true, b.handleCommand(s, update.Message)
}

func (b *bot) handleCommand(s *session, msg *tbot.Message) error {
	cmd := msg.Command()
	args := msg.CommandArguments()

	switch cmd {
	case StartCmd:
		return b.handleStartCommand(msg.Chat.ID, args)
	case ListDevicesCmd:
		return b.handleListDevicesCommand(s)
	case SelectAsDefaultCmd:
		return b.handleSelectAsDefaultCommand(s)
	case ResetCmd:
		return b.handleResetCommand(s)
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
			log.WithError(err).Errorf("Error occurred decoding base64 string `%v`", decoded)
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
Authentication is done using Yandex.OAuth. I will never ask you for login or password.
	`
	b.send(chatId, text)
	b.send(chatId, b.yaClient.getOAuthUrl())

	return nil
}

func (b *bot) handleListDevicesCommand(s *session) error {
	devices, err := b.getYandexStations(s)
	if err != nil {
		return err
	}

	iotInfo, _ := b.yaClient.getYandexSmartHomeInfo(s)

	msgText := b.formatYandexStationsMessage(s, devices, iotInfo.Rooms, iotInfo.Households)
	b.send(s.chatId, msgText)

	return nil
}

func (b *bot) getYandexStations(s *session) ([]device, error) {
	devices, err := b.yaClient.getYandexStations(s)
	if err != nil {
		return nil, NewBotError("Could not get list of registered devices. Please, try again.")
	}

	if len(devices) == 0 {
		return nil, NewBotError("I didn't find any yandex stations. Are they configured properly?")
	}

	return devices, nil
}

func (b *bot) formatYandexStationsMessage(s *session, devices []device, rooms []room, households []household) string {
	strs := b.yandexStationsToString(s, devices, rooms, households)

	buf := bytes.Buffer{}
	for i, v := range strs {
		buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, v))
	}

	return buf.String()
}

func (b *bot) yandexStationsToString(s *session, devices []device, rooms []room, households []household) []string {
	var lines = make([]string, 0)

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Id < devices[j].Id
	})
	for _, d := range devices {
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

		strFormat := "%s - %s - %s"
		if s.defaultDevice != nil && s.defaultDevice.Id == d.Id {
			strFormat = "Default: %s - %s - %s"
		}

		lines = append(lines, fmt.Sprintf(strFormat, h.Name, r.Name, d.Name))
	}

	return lines
}

func (b *bot) handleSelectAsDefaultCommand(s *session) error {
	devices, err := b.getYandexStations(s)
	if err != nil {
		return err
	}

	iotInfo, _ := b.yaClient.getYandexSmartHomeInfo(s)

	strs := b.yandexStationsToString(s, devices, iotInfo.Rooms, iotInfo.Households)

	rows := make([][]tbot.InlineKeyboardButton, 0)
	for i, s := range strs {
		data := fmt.Sprintf("%s:%s", SelectAsDefaultCallback, devices[i].Id)
		rows = append(rows, tbot.NewInlineKeyboardRow(tbot.NewInlineKeyboardButtonData(s, data)))
	}
	keyboard := tbot.NewInlineKeyboardMarkup(rows...)

	msg := tbot.NewMessage(s.chatId, "Please, select the station you want to make the default one.")
	msg.ReplyMarkup = keyboard

	//goland:noinspection GoUnhandledErrorResult
	b.api.Send(msg)

	return nil
}

func (b *bot) handleResetCommand(s *session) error {
	b.sessionProvider.Delete(s.chatId)

	ck1 := fmt.Sprintf("%d_%s", s.chatId, "iotuserinfo")
	b.cacheProvider.Delete(ck1)

	b.send(s.chatId, "Session has been reset successfully.")

	return nil
}

func (b *bot) handleSelectAsDefaultCommandCallback(s *session, deviceId string) {
	devices, err := b.yaClient.getYandexStations(s)
	if err != nil {
		log.WithError(err).Error("Could not process callback. Error occurred trying to get yandex stations")
		return
	}

	for _, d := range devices {
		if d.Id == deviceId {
			ns := NewSessionWithDevice(s, &d)
			b.sessionProvider.SaveOrUpdate(ns)

			b.send(s.chatId, fmt.Sprintf("Nice! Device `%s` is selected as default.", d.Name))
			return
		}
	}

	b.send(s.chatId, "Selected device is not currently available. Please, try again later.")
}

func (b *bot) handleOneTimePlayMediaCallback(s *session, replyToMessage *tbot.Message, deviceId string) {
	r, _ := regexp.Compile(URLRegexPattern)
	url := string(r.Find([]byte(replyToMessage.Text)))
	// It is guaranteed that at this point
	// we might have only youtube link
	url = reformatYouTubeUrl(url)

	log.Infof(url)

	devices, err := b.yaClient.getYandexStations(s)
	if err != nil {
		return
	}

	for _, d := range devices {
		if d.Id == deviceId {
			b.yaClient.playMedia(s, &d, url)
		}
	}
}

func (b *bot) isAuthorizationRequired(upd *tbot.Update) bool {
	if upd.Message != nil && upd.Message.Command() == StartCmd {
		return false
	}

	return true
}

func (b *bot) send(chatId int64, text string) {
	msg := tbot.NewMessage(chatId, text)
	_, err := b.api.Send(msg)
	if err != nil {
		log.WithError(err).Errorf("Error occurred while trying to send the message to chat %v", chatId)
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

	text := fmt.Sprintf(`
Something went wrong. Please, try again.
If the issue persists, try /%s-ting
`, ResetCmd)
	b.send(chatId, text)
}
