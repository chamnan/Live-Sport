package handlers

import (
	"log"

	"kbs-iptv/services"

	"github.com/gofiber/fiber/v2"
)

func GetStream(c *fiber.Ctx) error {

	matchId := c.Params("matchId")
	log.Printf("[API] GET /api/stream/%s", matchId)

	streamUrl, err := services.GetStream(matchId)
	if err != nil {
		log.Printf("[ERROR] GET /api/stream/%s → %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if streamUrl == "" {
		log.Printf("[API] GET /api/stream/%s → 200 (no stream)", matchId)
	} else {
		log.Printf("[API] GET /api/stream/%s → 200 (stream resolved)", matchId)
	}

	return c.JSON(fiber.Map{
		"streamUrl": streamUrl,
	})
}
