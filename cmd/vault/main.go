// ASGARD Security Vault Service
//
// Provides secure storage for sensitive access codes, API keys, and proprietary
// algorithm configurations with FIDO2 authentication support.
//
// DO-178C DAL-B Compliant: All access is audited and encrypted with AES-256-GCM.
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

	"github.com/asgard/pandora/internal/security/vault"
)

var (
	version = "1.0.0"
	build   = "dev"
)

func main() {
	// Command line flags
	httpAddr := flag.String("http", ":8094", "HTTP API address")
	storagePath := flag.String("storage", "./data/vault/secrets.enc", "Path to encrypted storage file")
	auditPath := flag.String("audit", "./data/vault/audit.log", "Path to audit log file")
	masterPassword := flag.String("password", "", "Master password (use env VAULT_MASTER_PASSWORD in production)")
	showVersion := flag.Bool("version", false, "Show version information")
	autoUnseal := flag.Bool("auto-unseal", false, "Automatically unseal vault on startup (dev mode only)")

	flag.Parse()

	if *showVersion {
		fmt.Printf("ASGARD Security Vault v%s (build: %s)\n", version, build)
		fmt.Println("\nCapabilities:")
		fmt.Println("  - AES-256-GCM encryption")
		fmt.Println("  - FIDO2/WebAuthn authentication")
		fmt.Println("  - AI-powered anomaly detection")
		fmt.Println("  - DO-178C DAL-B compliant audit logging")
		fmt.Println("\nSecurity Levels:")
		fmt.Println("  - public:     Basic authentication")
		fmt.Println("  - developer:  Developer credentials")
		fmt.Println("  - admin:      Admin privileges")
		fmt.Println("  - government: FIDO2 required")
		fmt.Println("  - military:   FIDO2 + biometric required")
		os.Exit(0)
	}

	// Get master password
	password := *masterPassword
	if password == "" {
		password = os.Getenv("VAULT_MASTER_PASSWORD")
	}

	if password == "" && !*autoUnseal {
		log.Println("[Vault] Warning: No master password provided. Vault will start sealed.")
		log.Println("[Vault] Use -password flag or VAULT_MASTER_PASSWORD environment variable.")
	}

	// Print banner
	printBanner()

	// Create vault configuration
	cfg := vault.DefaultVaultConfig()
	cfg.StoragePath = *storagePath
	cfg.AuditLogPath = *auditPath

	// Create vault
	v, err := vault.NewVault(cfg)
	if err != nil {
		log.Fatalf("[Vault] Failed to create vault: %v", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize vault if password provided
	if password != "" && *autoUnseal {
		if err := v.Initialize(ctx, password); err != nil {
			log.Fatalf("[Vault] Failed to initialize vault: %v", err)
		}
		log.Println("[Vault] Vault initialized and unsealed")
	} else {
		log.Println("[Vault] Vault started in sealed mode")
		log.Println("[Vault] POST to /vault/unseal with master_password to unseal")
	}

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("\n[Vault] Shutting down...")

		// Seal vault before exit
		if err := v.Seal(); err != nil {
			log.Printf("[Vault] Warning: Failed to seal vault: %v", err)
		}

		cancel()
	}()

	// Start HTTP server
	log.Printf("[Vault] Starting HTTP API on %s", *httpAddr)
	if err := vault.StartVaultServer(ctx, *httpAddr, v); err != nil {
		if err.Error() != "http: Server closed" {
			log.Fatalf("[Vault] Server error: %v", err)
		}
	}

	log.Println("[Vault] Shutdown complete")
}

func printBanner() {
	banner := `
╔═══════════════════════════════════════════════════════════════════╗
║                    ASGARD SECURITY VAULT                          ║
║                                                                   ║
║     AES-256-GCM Encryption | FIDO2 Authentication | AI Agent     ║
║                                                                   ║
║  Copyright 2026 Arobi. All Rights Reserved.                      ║
║  DO-178C DAL-B Compliant | CONFIDENTIAL - PROPRIETARY            ║
╚═══════════════════════════════════════════════════════════════════╝
`
	fmt.Println(banner)
}
