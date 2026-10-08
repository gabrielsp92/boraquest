// Command webapp starts the Boraquest HTTP API.
package main

import (
	"log"
	"os"

	"github.com/gabrielsp92/boraquest/back-end/cmd/webapp/routes"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/app/service"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/infrastructure/clock"
	"github.com/gabrielsp92/boraquest/back-end/internal/src/interface/http/controllers"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	// Infrastructure
	systemClock := clock.SystemClock{}

	// Application
	healthService := service.NewHealthService(systemClock, version)

	// Interface
	router := routes.NewRouter(routes.Controllers{
		Health: controllers.NewHealthController(healthService),
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("boraquest api listening on :%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
