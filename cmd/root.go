package cmd

import (
	"fmt"
	"github.com/etherlabsio/healthcheck"
	"log"
	"net/http"
	"os"
	"time"
)

func Execute() {
	setupServer()
}

func setupServer() {
	http.Handle("/health", healthcheck.Handler(
		healthcheck.WithTimeout(5*time.Second),
	))

	err := http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil)
	log.Fatalln(err)
}
