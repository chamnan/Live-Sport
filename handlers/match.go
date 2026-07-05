package handlers

import (
	"log"
	"strconv"
	"time"

	"kbs-iptv/services"

	"github.com/gofiber/fiber/v2"
)

func GetMatches(c *fiber.Ctx) error {

	now := time.Now()
	days := []string{
		now.AddDate(0, 0, -1).Format("2006-01-02"), // yesterday
		now.Format("2006-01-02"),                   // today
		now.AddDate(0, 0, 1).Format("2006-01-02"),  // tomorrow
	}

	// Optional filters: ?isLive=true&hasStream=true
	filterLive := c.Query("isLive")
	filterStream := c.Query("hasStream")

	log.Printf("[API] GET /api/matches → fetching %v (isLive=%q, hasStream=%q)", days, filterLive, filterStream)

	result := make(map[string]interface{})

	for _, day := range days {
		data, err := services.FetchMatches(day)
		if err != nil {
			log.Printf("[ERROR] GET /api/matches?day=%s → %v", day, err)
			result[day] = fiber.Map{"error": err.Error()}
			continue
		}

		// Apply isLive / hasStream filters if provided
		matches := data.Data
		if filterLive != "" {
			live, _ := strconv.ParseBool(filterLive)
			filtered := make([]services.Match, 0)
			for _, m := range matches {
				if m.IsLive == live {
					filtered = append(filtered, m)
				}
			}
			matches = filtered
		}
		if filterStream != "" {
			stream, _ := strconv.ParseBool(filterStream)
			filtered := make([]services.Match, 0)
			for _, m := range matches {
				if m.HasStream == stream {
					filtered = append(filtered, m)
				}
			}
			matches = filtered
		}

		result[day] = fiber.Map{
			"matches": matches,
			"meta":    data.Meta,
		}
	}

	log.Printf("[API] GET /api/matches → 200 OK (3 days)")
	return c.JSON(fiber.Map{
		"days":   days,
		"result": result,
	})
}
