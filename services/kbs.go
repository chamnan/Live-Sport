package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// Match represents a single fixture from the API.
type Match struct {
	ID                int         `json:"id"`
	ProxyAnimationURL string      `json:"proxyAnimationUrl"`
	Status            string      `json:"status"`
	StatusID          int         `json:"statusId"`
	Minute            interface{} `json:"minute"`
	IsLive            bool        `json:"isLive"`
	KickoffAt         string      `json:"kickoffAt"`
	HasStream         bool        `json:"hasStream"`
	Round             interface{} `json:"round"`
	Competition       Competition `json:"competition"`
	Home              TeamScore   `json:"home"`
	Away              TeamScore   `json:"away"`
}

type Competition struct {
	ID   int    `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}

type TeamScore struct {
	ID    int    `json:"id"`
	Slug  string `json:"slug"`
	Name  string `json:"name"`
	Logo  string `json:"logo"`
	Score int    `json:"score"`
}

// FixturesResponse is the top-level API response for fixtures.
type FixturesResponse struct {
	Data []Match `json:"data"`
	Meta Meta    `json:"meta"`
}

type Meta struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type StreamEntry struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	HlsURL string `json:"hlsUrl"`
}

type StreamsResponse struct {
	Data []StreamEntry `json:"data"`
	Meta struct {
		Available bool `json:"available"`
	} `json:"meta"`
}

func GetStream(matchId string) (string, error) {

	url := fmt.Sprintf("https://www.kbs388.com/api/football/fixtures/%s/streams", matchId)
	log.Printf("[UPSTREAM] GET %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data StreamsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("failed to parse streams: %w", err)
	}

	if len(data.Data) == 0 {
		log.Printf("[UPSTREAM] matchId=%s → no streams", matchId)
		return "", nil
	}

	hlsUrl := data.Data[0].HlsURL
	log.Printf("[UPSTREAM] matchId=%s → hlsUrl=%s", matchId, hlsUrl)

	return hlsUrl, nil
}

func FetchMatches(day string) (*FixturesResponse, error) {

	url := fmt.Sprintf(
		"https://www.kbs388.com/api/football/fixtures?date=%s",
		day,
	)
	log.Printf("[UPSTREAM] GET %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data FixturesResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse fixtures: %w", err)
	}

	log.Printf("[UPSTREAM] GET fixtures?date=%s → 200 (count=%d)", day, data.Meta.Count)

	return &data, nil
}
