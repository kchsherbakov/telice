package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type iotInfo struct {
	Status     string      `json:"status"`
	Message    string      `json:"message"`
	Rooms      []room      `json:"rooms"`
	Devices    []device    `json:"devices"`
	Households []household `json:"households"`
}

type room struct {
	Id          string   `json:"id"`
	Name        string   `json:"name"`
	HouseholdId string   `json:"household_id"`
	Devices     []string `json:"devices"`
}

type household struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type device struct {
	Id         string     `json:"id"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Room       string     `json:"room"`
	QuasarInfo quasarInfo `json:"quasar_info"`
}

type quasarInfo struct {
	Id       string `json:"device_id"`
	Platform string `json:"platform"`
}

type YandexClient struct {
	clientId      string
	cacheProvider CacheProvider
	httpClient    *http.Client
}

func NewYandexClient(clientId string, cacheProvider CacheProvider, httpClient *http.Client) *YandexClient {
	if httpClient == nil {
		log.Fatalln("Http client must not be null")
	}

	return &YandexClient{clientId, cacheProvider, httpClient}
}

func (y *YandexClient) getTokens(rawToken string) (*token, *token, error) {
	tokenInfo := strings.Split(rawToken, ":")
	accessToken := tokenInfo[0]
	expiresIn, _ := strconv.Atoi(tokenInfo[1])

	oauthToken := NewToken(accessToken, &expiresIn)

	csrfToken, err := y.getYandexCSRFToken(oauthToken.value)
	if err != nil {
		return nil, nil, err
	}

	return oauthToken, csrfToken, nil
}

func (y *YandexClient) getYandexCSRFToken(oauthToken string) (*token, error) {
	if oauthToken == "" {
		return nil, errors.New("yandex OAuth token is required to perform this action")
	}

	req, err := http.NewRequest(http.MethodGet, "https://frontend.vh.yandex.ru/csrf_token", nil)
	req.Header.Add("Authorization", fmt.Sprintf("OAuth %s", oauthToken))

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

func (y *YandexClient) getYandexSmartHomeInfo(s *session) (*iotInfo, error) {
	cacheKey := fmt.Sprintf("%d_%s", s.chatId, "iotuserinfo")
	val, found := y.cacheProvider.TryGet(cacheKey)
	if found {
		return val.(*iotInfo), nil
	}

	req, err := http.NewRequest(http.MethodGet, "https://api.iot.yandex.net/v1.0/user/info", nil)
	if err != nil {
		log.Print("Could not create new request", err)
		return nil, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("OAuth %s", s.oauthToken.value))

	resp, err := y.httpClient.Do(req)
	if err != nil {
		log.Println("Error occurred while requesting devices info", err)
		return nil, err
	}

	var dataResp = &iotInfo{}
	err = json.NewDecoder(resp.Body).Decode(dataResp)
	if err != nil {
		log.Println("Error occurred while decoding response body", err)
		return nil, err
	}

	if dataResp.Status != "ok" {
		log.Println(fmt.Sprintf("Request has completed with error status. Message: %s", dataResp.Message))
		return nil, errors.New("request has completed with error status")
	}

	y.cacheProvider.Save(cacheKey, dataResp)

	return dataResp, nil
}

func (y *YandexClient) getYandexStations(s *session) ([]device, error) {
	iotInfo, err := y.getYandexSmartHomeInfo(s)
	if err != nil {
		return nil, err
	}

	stations := make([]device, 0)
	for _, d := range iotInfo.Devices {
		if !strings.Contains(d.Type, YandexStationTypeSubstr) {
			continue
		}

		stations = append(stations, d)
	}

	return stations, nil
}

func (y *YandexClient) getOAuthUrl() string {
	return fmt.Sprintf("https://oauth.yandex.com/authorize?response_type=token&client_id=%v", y.clientId)
}
