// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

// Go version 1.20 or newer is required due to how it initializes the PRNG.
//go:build go1.20

package main

import (
	"log"

	"github.com/Team254/cheesy-arena-lite/config"
	"github.com/Team254/cheesy-arena-lite/field"
	"github.com/Team254/cheesy-arena-lite/version"
	"github.com/Team254/cheesy-arena-lite/web"
)

// Main entry point for the application.
func main() {
	cfg := config.Load()
	log.Printf("CyberArena %s", version.Version)

	arena, err := field.NewArena(cfg.DbPath, cfg)
	if err != nil {
		log.Fatalln("Error during startup: ", err)
	}

	// Start the web server in a separate goroutine.
	web := web.NewWeb(arena)
	go web.ServeWebInterface(cfg.Port)

	// Run the arena state machine in the main thread.
	arena.Run()
}
