package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// userAgent mimics a browser. The kbs388.com API returns a 404 for Go's
// default User-Agent ("Go-http-client/1.1"), which broke stream playback.
const userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

var httpClient = &http.Client{}

// getJSON fetches a URL with browser-like headers and returns the body bytes.
func getJSON(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", "https://www.kbs388.com/")
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

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

// UnmarshalJSON handles the case where the API returns "data" as an
// empty object {} instead of an empty array [] when no fixtures exist.
func (fr *FixturesResponse) UnmarshalJSON(b []byte) error {
	type Alias struct {
		Data json.RawMessage `json:"data"`
		Meta Meta            `json:"meta"`
	}
	var alias Alias
	if err := json.Unmarshal(b, &alias); err != nil {
		return err
	}

	fr.Meta = alias.Meta

	// Try as array first; if that fails, treat it as an empty object → empty slice
	if err := json.Unmarshal(alias.Data, &fr.Data); err != nil {
		// If data is an object (e.g. {}), default to empty slice
		fr.Data = []Match{}
	}

	return nil
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

func GetStreams(matchId string) ([]StreamEntry, error) {

	url := fmt.Sprintf("https://www.kbs388.com/api/football/fixtures/%s/streams", matchId)
	log.Printf("[UPSTREAM] GET %s", url)

	body, err := getJSON(url)
	if err != nil {
		return nil, err
	}

	var data StreamsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse streams: %w", err)
	}

	if len(data.Data) == 0 {
		log.Printf("[UPSTREAM] matchId=%s → no streams", matchId)
		return nil, nil
	}

	log.Printf("[UPSTREAM] matchId=%s → %d streams", matchId, len(data.Data))

	return data.Data, nil
}

func FetchMatches(day string) (*FixturesResponse, error) {

	url := fmt.Sprintf(
		"https://www.kbs388.com/api/football/fixtures?date=%s",
		day,
	)
	log.Printf("[UPSTREAM] GET %s", url)

	body, err := getJSON(url)
	if err != nil {
		return nil, err
	}

	var data FixturesResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse fixtures: %w", err)
	}

	log.Printf("[UPSTREAM] GET fixtures?date=%s → 200 (count=%d)", day, data.Meta.Count)

	return &data, nil
}
