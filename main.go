package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "2006-01-02 15:04:05", NoColor: false})
	var configPath string

	flag.StringVar(&configPath, "config", "", "Path to the configuration file")
	flag.Parse()

	initConfig(configPath)

	http.HandleFunc(Pathhook, hookData)

	address := os.Getenv("SERVER_ADDRESS")
	if address == "" {
		address = "127.0.0.1"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "42135"
	}

	serverAddr := address + ":" + port
	log.Info().Msg("Starting server on " + serverAddr)
	err := http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start server")
	}
}
