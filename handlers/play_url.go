package handlers

import (
	"log"
	"net/url"

	"github.com/gofiber/fiber/v2"
)

func BuildPlayURL(c *fiber.Ctx) error {

	stream := c.Query("u")
	matchId := c.Query("matchId")
	home := c.Query("home")
	away := c.Query("away")

	log.Printf("[API] GET /api/play-url?matchId=%s&home=%s&away=%s", matchId, home, away)

	if stream == "" {
		log.Printf("[ERROR] GET /api/play-url → missing stream url")
		return c.Status(400).JSON(fiber.Map{
			"error": "missing stream url",
		})
	}

	finalURL := "https://www.kbs388.com/api/kbs-hls?u=" +
		url.QueryEscape(stream) +
		"&matchId=" + matchId +
		"&home=" + url.QueryEscape(home) +
		"&away=" + url.QueryEscape(away)

	log.Printf("[API] GET /api/play-url → 200 (resolved)")
	return c.JSON(fiber.Map{
		"url": finalURL,
	})
}
