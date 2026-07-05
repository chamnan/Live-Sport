package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func BuildPlayURL(c *fiber.Ctx) error {

	hlsUrl := c.Query("u")
	matchId := c.Query("matchId")

	log.Printf("[API] GET /api/play-url?matchId=%s", matchId)

	if hlsUrl == "" {
		log.Printf("[ERROR] GET /api/play-url → missing hls url")
		return c.Status(400).JSON(fiber.Map{
			"error": "missing hls url",
		})
	}

	log.Printf("[API] GET /api/play-url → 200 (resolved)")
	return c.JSON(fiber.Map{
		"url": hlsUrl,
	})
}
