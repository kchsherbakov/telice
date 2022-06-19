package main

const (
	HostEnv          = "HOST"
	PortEnv          = "PORT"
	TelegramBotToken = "TELEGRAM_BOT_TOKEN"
	YandexClientId   = "YANDEX_CLIENT_ID"

	StartCmd           = "start"
	ListDevicesCmd     = "listdevices"
	SelectAsDefaultCmd = "selectasdefault"
	ResetCmd           = "reset"

	SelectAsDefaultCallback  = "sad"
	OneTimePlayMediaCallback = "otp"

	YandexStationTypeSubstr = "yandex.station"

	URLRegexPattern = "(?:(?:https?):\\/\\/|\\b(?:[a-z\\d]+\\.))(?:(?:[^\\s()<>]+|\\((?:[^\\s()<>]+|(?:\\([^\\s()<>]+\\)))?\\))+(?:\\((?:[^\\s()<>]+|(?:\\(?:[^\\s()<>]+\\)))?\\)|[^\\s`!()\\[\\]{};:'\".,<>?«»“”‘’]))?"
)
