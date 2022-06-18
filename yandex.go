package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type iotInfoResponse struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Devices []device `json:"devices"`
}

type device struct {
	Id         string     `json:"id"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	QuasarInfo quasarInfo `json:"quasar_info"`
}

type quasarInfo struct {
	Id       string `json:"device_id"`
	Platform string `json:"platform"`
}

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

func (y *YandexClient) getYandexStations() ([]device, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.iot.yandex.net/v1.0/user/info", nil)
	if err != nil {
		log.Print("Could not create new request", err)
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("OAuth %s", y.oauthToken.value))

	resp, err := y.httpClient.Do(req)
	if err != nil {
		log.Println("Error occurred while requesting devices info", err)
		return nil, err
	}

	var dataResp = &iotInfoResponse{}
	err = json.NewDecoder(resp.Body).Decode(dataResp)
	if err != nil {
		log.Println("Error occurred while decoding response body", err)
		return nil, err
	}

	if dataResp.Status != "ok" {
		log.Println(fmt.Sprintf("Request has completed with error status. Message: %s", dataResp.Message))
		return nil, errors.New("request has completed with error status")
	}

	stations := make([]device, 0)
	for _, d := range dataResp.Devices {
		if !strings.Contains(d.Type, YandexStationTypeSubstr) {
			continue
		}

		stations = append(stations, d)
	}

	return stations, nil
}
