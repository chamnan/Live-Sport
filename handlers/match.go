package handlers

import (
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"

	"kbs-iptv/services"
)

const dateLayout = "2006-01-02"

// GetMatches returns yesterday's, today's and tomorrow's fixtures keyed by day.
// Optional filters: ?isLive=true&hasStream=true
func GetMatches(c *fiber.Ctx) error {
	now := time.Now()
	days := []string{
		now.AddDate(0, 0, -1).Format(dateLayout), // yesterday
		now.Format(dateLayout),                   // today
		now.AddDate(0, 0, 1).Format(dateLayout),  // tomorrow
	}

	filterLive := c.Query("isLive")
	filterStream := c.Query("hasStream")

	log.Printf("[API] GET /api/matches → fetching %v (isLive=%q, hasStream=%q)", days, filterLive, filterStream)

	// Fetch all days in parallel; each upstream call can take ~10s.
	type fetched struct {
		data *services.FixturesResponse
		err  error
	}
	responses := make([]fetched, len(days))
	var wg sync.WaitGroup
	for i, day := range days {
		wg.Go(func() {
			data, err := services.FetchMatches(day)
			responses[i] = fetched{data, err}
		})
	}
	wg.Wait()

	result := make(map[string]any, len(days))

	for i, day := range days {
		data, err := responses[i].data, responses[i].err
		if err != nil {
			log.Printf("[ERROR] GET /api/matches?day=%s → %v", day, err)
			result[day] = fiber.Map{"error": err.Error()}
			continue
		}

		matches := data.Data
		if filterLive != "" {
			live, _ := strconv.ParseBool(filterLive)
			matches = filterMatches(matches, func(m services.Match) bool { return m.IsLive == live })
		}
		if filterStream != "" {
			stream, _ := strconv.ParseBool(filterStream)
			matches = filterMatches(matches, func(m services.Match) bool { return m.HasStream == stream })
		}

		result[day] = fiber.Map{
			"matches": matches,
			"meta":    data.Meta,
		}
	}

	log.Printf("[API] GET /api/matches → 200 OK (%d days)", len(days))
	return c.JSON(fiber.Map{
		"days":   days,
		"result": result,
	})
}

// filterMatches returns the matches for which keep returns true.
func filterMatches(matches []services.Match, keep func(services.Match) bool) []services.Match {
	filtered := make([]services.Match, 0, len(matches))
	for _, m := range matches {
		if keep(m) {
			filtered = append(filtered, m)
		}
	}
	return filtered
}
