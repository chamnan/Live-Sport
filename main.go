package main

import (
	"embed"
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"

	"kbs-iptv/handlers"
)

//go:embed web/*
var webFS embed.FS

func main() {

	app := fiber.New()

	// FIX CORS (frontend safe)
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "*",
	}))

	// APIs
	app.Get("/api/matches", handlers.GetMatches)
	app.Get("/api/stream/:matchId", handlers.GetStream)
	app.Get("/api/play-url", handlers.BuildPlayURL)

	// static frontend (embedded — works regardless of CWD)
	app.Use("/", filesystem.New(filesystem.Config{
		Root:       http.FS(webFS),
		PathPrefix: "web",
		Browse:     false,
	}))

	log.Fatal(app.Listen(":8090"))
}
