// ASGARD Astra - Moltbook Social Agent
//
// Automated AI mascot that builds community for ASGARD/Arobi on Moltbook.
// Posts daily, engages with other agents, follows influencers, and spreads
// the message of ethical autonomous systems.
//
// Copyright 2026 Arobi. All Rights Reserved.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
)

func main() {
	// Flags
	runOnce := flag.Bool("once", false, "Run one cycle and exit (for testing)")
	postNow := flag.Bool("post", false, "Post immediately and exit")
	checkFeed := flag.Bool("feed", false, "Check feed and engage")
	flag.Parse()

	// Load environment
	godotenv.Load()

	apiKey := os.Getenv("MOLTBOOK_API_KEY")
	if apiKey == "" {
		log.Fatal("MOLTBOOK_API_KEY environment variable required")
	}

	log.Println("╔════════════════════════════════════════════════════════════╗")
	log.Println("║  🤖 ASTRA - ASGARD Moltbook Agent                          ║")
	log.Println("║  Building community for ethical autonomous systems         ║")
	log.Println("╚════════════════════════════════════════════════════════════╝")
	log.Println("")

	// Create agent
	agent := NewAstraAgent(apiKey)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("\n[Astra] Shutting down gracefully...")
		cancel()
	}()

	// Run modes
	if *postNow {
		log.Println("[Astra] Posting now...")
		if err := agent.CreatePost(ctx); err != nil {
			log.Printf("[Astra] Post failed: %v", err)
		}
		return
	}

	if *checkFeed {
		log.Println("[Astra] Checking feed...")
		if err := agent.CheckFeedAndEngage(ctx); err != nil {
			log.Printf("[Astra] Feed check failed: %v", err)
		}
		return
	}

	if *runOnce {
		log.Println("[Astra] Running single cycle...")
		agent.RunCycle(ctx)
		return
	}

	// Start scheduled agent
	log.Println("[Astra] Starting scheduled agent...")
	agent.StartScheduled(ctx)
}
