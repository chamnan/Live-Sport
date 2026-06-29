# kbs-iptv

IPTV sports streaming player with a Go backend and web frontend.

## Stack

- **Backend:** Go + [Fiber](https://gofiber.io/) v2
- **Frontend:** Vanilla HTML/CSS/JS + [ArtPlayer](https://artplayer.org/) + [hls.js](https://github.com/video-dev/hls.js)

## API

| Endpoint | Description |
|---|---|
| `GET /api/matches` | List all matches grouped by day |
| `GET /api/stream/:matchId` | Get stream URL for a match |
| `GET /api/play-url?u=...&matchId=...&home=...&away=...` | Build final HLS play URL |

## Setup

```bash
go mod tidy
```

## Run

```bash
go run .
```

Then open `http://localhost:8090`.

## Build

```bash
go build -o kbs-iptv .
./kbs-iptv
```
