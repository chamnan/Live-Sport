package handlers

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"kbs-iptv/services"
)

// StreamInfo is a playable stream as returned to the frontend.
type StreamInfo struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	StreamURL string `json:"streamUrl"`
}

// GetStream returns the available streams for a match, with each HLS URL
// rewritten to go through the same-origin /hls/ proxy.
func GetStream(c *fiber.Ctx) error {
	matchID := c.Params("matchId")
	log.Printf("[API] GET /api/stream/%s", matchID)

	streams, err := services.GetStreams(matchID)
	if err != nil {
		log.Printf("[ERROR] GET /api/stream/%s → %v", matchID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
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

	log.Printf("[API] GET /api/stream/%s → 200 (%d streams)", matchID, len(out))
	return c.JSON(fiber.Map{
		"streams": out,
	})
}
