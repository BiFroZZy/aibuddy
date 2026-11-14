package main

import (
	"aibuddy/cmd/server"
	"log/slog"
	"os"
)

func main() {
	srv, err := server.New()
	if err != nil{
		slog.Error("failed to initialize server", "error", err)
		os.Exit(1)
	}

	if err := srv.Start(); err != nil{
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

