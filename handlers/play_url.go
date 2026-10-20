package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

// BuildPlayURL returns the final HLS play URL for a match.
func BuildPlayURL(c *fiber.Ctx) error {
	hlsURL := c.Query("u")
	matchID := c.Query("matchId")

	log.Printf("[API] GET /api/play-url?matchId=%s", matchID)

	if hlsURL == "" {
		log.Printf("[ERROR] GET /api/play-url → missing hls url")
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "missing hls url",
		})
	}

	log.Printf("[API] GET /api/play-url → 200 (resolved)")
	return c.JSON(fiber.Map{
		"url": hlsURL,
	})
}
