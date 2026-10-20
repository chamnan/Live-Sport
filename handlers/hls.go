package handlers

import (
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"kbs-iptv/services"
)

// upstreamHLSBase is the host that serves HLS playlists (master + media).
// These playlists are returned WITHOUT Access-Control-Allow-Origin headers,
// which blocks hls.js in the browser. We proxy them through this server and
// rewrite internal playlist URLs back to the local /hls/ path.
const upstreamHLSBase = "https://football-backend-mlcakheg.on-forge.com/v1/football/hls/"

// hlsClient fetches playlists from upstreamHLSBase. The timeout keeps a hung
// upstream from tying up the proxy handler.
var hlsClient = &http.Client{Timeout: 20 * time.Second}

// ProxyHLSURL rewrites a direct upstream HLS URL into a same-origin proxy URL
// so the browser only ever talks to this server for playlist requests.
func ProxyHLSURL(raw, base string) string {
	if rest, ok := strings.CutPrefix(raw, upstreamHLSBase); ok {
		return strings.TrimRight(base, "/") + "/hls/" + rest
	}
	return raw
}

// ProxyHLS proxies HLS playlists from the upstream stream backend.
func ProxyHLS(c *fiber.Ctx) error {
	suffix := strings.TrimPrefix(c.Path(), "/hls/")
	if suffix == "" {
		return c.Status(fiber.StatusBadRequest).SendString("missing hls path")
	}

	upstreamURL := upstreamHLSBase + suffix
	if qs := c.Request().URI().QueryString(); len(qs) > 0 {
		upstreamURL += "?" + string(qs)
	}

	req, err := http.NewRequestWithContext(c.Context(), http.MethodGet, upstreamURL, nil)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
	}
	req.Header.Set("User-Agent", services.UserAgent)
	req.Header.Set("Referer", services.Referer)

	resp, err := hlsClient.Do(req)
	if err != nil {
		log.Printf("[HLS] proxy error %s: %v", upstreamURL, err)
		return c.Status(fiber.StatusBadGateway).SendString("upstream error")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).SendString("read error")
	}

	if resp.StatusCode != http.StatusOK {
		c.Set(fiber.HeaderContentType, fiber.MIMETextPlainCharsetUTF8)
		return c.Status(resp.StatusCode).Send(body)
	}

	// Playlists need their internal absolute URLs rewritten; segments (binary)
	// are forwarded as-is. Segment hosts (liveplay1.camel4.live) already allow
	// CORS *, so hls.js can fetch those directly.
	contentType := resp.Header.Get(fiber.HeaderContentType)
	isPlaylist := strings.Contains(contentType, "mpegurl") ||
		strings.HasPrefix(string(body), "#EXTM3U")

	if isPlaylist {
		c.Set(fiber.HeaderContentType, "application/vnd.apple.mpegurl")
		return c.SendString(rewriteHLSPlaylist(string(body)))
	}

	if contentType == "" {
		contentType = fiber.MIMEOctetStream
	}
	c.Set(fiber.HeaderContentType, contentType)
	return c.Send(body)
}

// rewriteHLSPlaylist rewrites absolute upstream HLS URLs back to the local
// /hls/ proxy so the browser only ever talks to this server for playlists.
func rewriteHLSPlaylist(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if rest, ok := strings.CutPrefix(trimmed, upstreamHLSBase); ok {
			lines[i] = "/hls/" + rest
		}
	}
	return strings.Join(lines, "\n")
}
