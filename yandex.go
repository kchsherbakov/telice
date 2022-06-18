package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type YandexClient struct {
	id         string
	oauthToken *token
	csrfToken  *token
	httpClient *http.Client
}

func NewYandexClient(httpClient *http.Client) *YandexClient {
	if httpClient == nil {
		log.Fatalln("Http client must not be null")
	}

	return &YandexClient{os.Getenv(YandexClientId), nil, nil, httpClient}
}

func (y *YandexClient) SetupTokens(rawToken string) error {
	tokenInfo := strings.Split(rawToken, ":")
	accessToken := tokenInfo[0]
	expiresIn, _ := strconv.Atoi(tokenInfo[1])

	y.oauthToken = NewToken(accessToken, &expiresIn)

	csrfToken, err := y.getYandexCSRFToken()
	if err != nil {
		return err
	}

	y.csrfToken = csrfToken

	return nil
}

func (y *YandexClient) getYandexCSRFToken() (*token, error) {
	if y.oauthToken == nil {
		return nil, errors.New("yandex OAuth token is required to perform this action")
	}

	req, err := http.NewRequest(http.MethodGet, "https://frontend.vh.yandex.ru/csrf_token", nil)
	req.Header.Add("Authorization", fmt.Sprintf("OAuth %s", y.oauthToken.value))

	resp, err := y.httpClient.Do(req)
	if err != nil {
		log.Println("Could not get yandex csrf token", err)
		return nil, err
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return NewToken(string(tokenBytes), nil), nil
}
