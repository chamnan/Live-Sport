package handlers

import (
	"log"
	"time"

	"kbs-iptv/services"

	"github.com/gofiber/fiber/v2"
)

func GetMatches(c *fiber.Ctx) error {

	now := time.Now()
	days := []string{
		now.AddDate(0, 0, -1).Format("20060102"), // yesterday
		now.Format("20060102"),                   // today
		now.AddDate(0, 0, 1).Format("20060102"),  // tomorrow
	}

	log.Printf("[API] GET /api/matches → fetching %v", days)

	result := make(map[string]interface{})

	for _, day := range days {
		data, err := services.FetchMatches(day)
		if err != nil {
			log.Printf("[ERROR] GET /api/matches?day=%s → %v", day, err)
			result[day] = fiber.Map{"error": err.Error()}
			continue
		}
		result[day] = data
	}

	log.Printf("[API] GET /api/matches → 200 OK (3 days)")
	return c.JSON(fiber.Map{
		"days":   days,
		"result": result,
	})
}
