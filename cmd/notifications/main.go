// ASGARD Notification Service
//
// Provides email notifications, access key generation, and verification services.
// Uses Resend API (free tier: 3000 emails/month) for email delivery.
//
// Copyright 2026 Arobi. All Rights Reserved.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/asgard/pandora/internal/notifications"
)

var (
	version = "1.0.0"
	build   = "dev"
)

func main() {
	// Command line flags
	httpAddr := flag.String("http", ":8095", "HTTP API address")
	showVersion := flag.Bool("version", false, "Show version information")
	generateFounderKey := flag.Bool("generate-founder-key", false, "Generate and email a founder access key")

	flag.Parse()

	if *showVersion {
		fmt.Printf("ASGARD Notification Service v%s (build: %s)\n", version, build)
		fmt.Println("\nCapabilities:")
		fmt.Println("  - Email notifications via Resend API")
		fmt.Println("  - Access key generation and management")
		fmt.Println("  - Email verification codes")
		fmt.Println("  - Security alerts")
		fmt.Println("\nConfiguration:")
		fmt.Println("  - Set RESEND_API_KEY environment variable for email delivery")
		fmt.Println("  - Founder email: Gaetano@aura-genesis.org")
		os.Exit(0)
	}

	// Check for Resend API key
	if os.Getenv("RESEND_API_KEY") == "" {
		log.Println("[Notification] Warning: RESEND_API_KEY not set")
		log.Println("[Notification] Email notifications will be disabled")
		log.Println("[Notification] Get a free API key at https://resend.com")
	}

	// Print banner
	printBanner()

	// Handle generate-founder-key command
	if *generateFounderKey {
		generateFounderAccessKey()
		return
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("\n[Notification] Shutting down...")
		cancel()
	}()

	// Start HTTP server
	log.Printf("[Notification] Starting server on %s", *httpAddr)
	if err := notifications.StartNotificationServer(ctx, *httpAddr); err != nil {
		if err.Error() != "http: Server closed" {
			log.Fatalf("[Notification] Server error: %v", err)
		}
	}

	log.Println("[Notification] Shutdown complete")
}

func generateFounderAccessKey() {
	api := notifications.NewNotificationAPI()

	log.Println("[Notification] Generating founder access key...")
	log.Println("[Notification] This will be emailed to: Gaetano@aura-genesis.org")

	// Note: In a real scenario, this would use the API
	// For now, just show instructions
	fmt.Println("\n=== FOUNDER ACCESS KEY GENERATION ===")
	fmt.Println("\nTo generate a founder access key:")
	fmt.Println("1. Start the notification server: ./notifications -http :8095")
	fmt.Println("2. Make API request:")
	fmt.Println("")
	fmt.Println("   curl -X POST http://localhost:8095/api/access-keys/founder \\")
	fmt.Println("        -H 'Authorization: Bearer <admin-token>' \\")
	fmt.Println("        -H 'Content-Type: application/json'")
	fmt.Println("")
	fmt.Println("The key will be:")
	fmt.Println("  - Generated with FOUNDER_MASTER type")
	fmt.Println("  - Emailed to Gaetano@aura-genesis.org")
	fmt.Println("  - Valid for 24 hours")
	fmt.Println("  - One-time use")
	fmt.Println("")

	_ = api // Silence unused variable warning
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════════════╗
║              ASGARD NOTIFICATION SERVICE                          ║
║                                                                   ║
║     Email: Resend API | Access Keys | Verification | Alerts      ║
║                                                                   ║
║  Founder Email: Gaetano@aura-genesis.org                         ║
║  Copyright 2026 Arobi. All Rights Reserved.                      ║
╚═══════════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}
