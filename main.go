package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"github.com/woodchen/docker-mirror-go/internal/handler"
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	// Root redirect
	r.GET("/", func(c *gin.Context) {
		c.Redirect(301, "https://onepage.czl.net/tools/docker_mirror.html")
	})

	// Registry proxy
	r.Any("/v2/*path", handler.HandleRegistryRequest)

	zlog.Info().Str("port", port).Msg("Starting Docker registry proxy server")
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
