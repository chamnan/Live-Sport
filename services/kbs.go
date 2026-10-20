package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	// UserAgent mimics a browser. The kbs388.com API returns a 404 for Go's
	// default User-Agent ("Go-http-client/1.1"), which broke stream playback.
	UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

	// Referer is sent with upstream requests so they look like they come from
	// the kbs388.com site.
	Referer = "https://www.kbs388.com/"

	apiBase = "https://www.kbs388.com/api/football"
)

// httpClient talks to the kbs388.com API. The timeout keeps a hung upstream
// from hanging our handlers; fixtures requests can take ~10s.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// getJSON fetches a URL with browser-like headers and returns the body bytes.
// Non-200 responses are returned as errors.
func getJSON(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Referer", Referer)
	req.Header.Set("Accept", "application/json, text/plain, */*")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 200))
		return nil, fmt.Errorf("upstream %s returned %s: %s", url, resp.Status, strings.TrimSpace(string(snippet)))
	}

	return io.ReadAll(resp.Body)
}

// Match represents a single fixture from the API.
type Match struct {
	ID                int         `json:"id"`
	ProxyAnimationURL string      `json:"proxyAnimationUrl"`
	Status            string      `json:"status"`
	StatusID          int         `json:"statusId"`
	Minute            any         `json:"minute"`
	IsLive            bool        `json:"isLive"`
	KickoffAt         string      `json:"kickoffAt"`
	HasStream         bool        `json:"hasStream"`
	Round             any         `json:"round"`
	Competition       Competition `json:"competition"`
	Home              TeamScore   `json:"home"`
	Away              TeamScore   `json:"away"`
}

// Competition is the league or cup a match belongs to.
type Competition struct {
	ID   int    `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}

// TeamScore is one side of a match and its current score.
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

	// Try as array first; if data is an object (e.g. {}), default to an empty slice.
	if err := json.Unmarshal(alias.Data, &fr.Data); err != nil {
		fr.Data = []Match{}
	}
	return nil
}

// Meta describes a fixtures response.
type Meta struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// StreamEntry is a single upstream stream for a match.
type StreamEntry struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	HlsURL string `json:"hlsUrl"`
}

// StreamsResponse is the top-level API response for a match's streams.
type StreamsResponse struct {
	Data []StreamEntry `json:"data"`
	Meta struct {
		Available bool `json:"available"`
	} `json:"meta"`
}

// GetStreams fetches the available streams for a match.
func GetStreams(matchID string) ([]StreamEntry, error) {
	url := fmt.Sprintf("%s/fixtures/%s/streams", apiBase, matchID)
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
		log.Printf("[UPSTREAM] matchId=%s → no streams", matchID)
		return nil, nil
	}

	log.Printf("[UPSTREAM] matchId=%s → %d streams", matchID, len(data.Data))
	return data.Data, nil
}

// FetchMatches fetches the fixtures for a day (YYYY-MM-DD).
func FetchMatches(day string) (*FixturesResponse, error) {
	url := fmt.Sprintf("%s/fixtures?date=%s", apiBase, day)
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
