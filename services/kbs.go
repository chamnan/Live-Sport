package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func GetStream(matchId string) (string, error) {

	url := "https://www.kbs388.com/api/kbs/sports_live/loadAnchorsByMatchId?matchId=" + matchId
	log.Printf("[UPSTREAM] GET %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// detail is a single object: { anchorIds, streams[], robot }
	detail, ok := data["detail"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("no 'detail' object in response")
	}

	// streams is always an array (can be empty)
	streamsRaw, ok := detail["streams"]
	if !ok {
		log.Printf("[UPSTREAM] matchId=%s → no streams key", matchId)
		return "", nil
	}

	streams, ok := streamsRaw.([]interface{})
	if !ok || len(streams) == 0 {
		log.Printf("[UPSTREAM] matchId=%s → empty streams", matchId)
		return "", nil
	}

	// Grab the first stream entry
	firstStream, ok := streams[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected stream entry type")
	}

	streamUrl, _ := firstStream["streamUrl"].(string)
	log.Printf("[UPSTREAM] matchId=%s → streamUrl=%s", matchId, streamUrl)

	return streamUrl, nil
}

func FetchMatches(day string) (map[string]interface{}, error) {

	url := fmt.Sprintf(
		"https://www.kbs388.com/api/kbs/sports_live/home_match?day=%s",
		day,
	)
	log.Printf("[UPSTREAM] GET %s", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse matches: %w", err)
	}

	log.Printf("[UPSTREAM] GET home_match?day=%s → 200 (len=%d)", day, len(body))

	return data, nil
}
