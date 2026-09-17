package handlers

import (
	"log"

	"kbs-iptv/services"

	"github.com/gofiber/fiber/v2"
)

type StreamInfo struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	StreamURL string `json:"streamUrl"`
}

func GetStream(c *fiber.Ctx) error {

	matchId := c.Params("matchId")
	log.Printf("[API] GET /api/stream/%s", matchId)

	streams, err := services.GetStreams(matchId)
	if err != nil {
		log.Printf("[ERROR] GET /api/stream/%s → %v", matchId, err)
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	if len(streams) == 0 {
		log.Printf("[API] GET /api/stream/%s → 200 (no stream)", matchId)
		return c.JSON(fiber.Map{
			"streams": []StreamInfo{},
		})
	}

	// Rewrite each upstream HLS URL into a same-origin proxy URL so the browser
	// can fetch playlists without hitting CORS restrictions.
	base := c.Protocol() + "://" + string(c.Request().Host())
	out := make([]StreamInfo, 0, len(streams))
	for _, s := range streams {
		out = append(out, StreamInfo{
			ID:        s.ID,
			Label:     s.Label,
			StreamURL: ProxyHLSURL(s.HlsURL, base),
		})
	}
	log.Printf("[API] GET /api/stream/%s → 200 (%d streams)", matchId, len(out))

	return c.JSON(fiber.Map{
		"streams": out,
	})
}
