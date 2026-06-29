package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"kbs-iptv/handlers"
)

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

	// static frontend
	app.Static("/", "./web")

	log.Fatal(app.Listen(":8090"))
}
