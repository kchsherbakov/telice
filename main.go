package main

import (
	"fmt"
	"github.com/etherlabsio/healthcheck"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

var log *logrus.Logger

func main() {
	initConfig()
	checkEnv()

	go setupServer()
	runBot()
}

func initConfig() {
	log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	log.SetReportCaller(true)
}

func checkEnv() {
	var names = make([]string, 0)

	v := os.Getenv(HostEnv)
	if v == "" {
		names = append(names, HostEnv)
	}
	v = os.Getenv(PortEnv)
	if v == "" {
		names = append(names, PortEnv)
	}
	v = os.Getenv(TelegramBotToken)
	if v == "" {
		names = append(names, TelegramBotToken)
	}
	v = os.Getenv(YandexClientId)
	if v == "" {
		names = append(names, YandexClientId)
	}

	if len(names) > 0 {
		log.Fatalf("One or more ENV variables are not set. Please, check the following variables: %v", names)
	}
}

func setupServer() {
	http.Handle("/health", healthcheck.Handler(
		healthcheck.WithTimeout(5*time.Second),
	))
	http.Handle("/", http.FileServer(http.Dir('.')))

	addr := fmt.Sprintf("%s:%s", os.Getenv(HostEnv), os.Getenv(PortEnv))
	err := http.ListenAndServe(addr, nil)
	log.Fatalln(err)
}

func runBot() {
	b := NewBot(os.Getenv(TelegramBotToken), os.Getenv(YandexClientId))
	b.Run()
}
