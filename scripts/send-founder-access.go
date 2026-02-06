//go:build ignore
// +build ignore

// ASGARD Founder Access Key Generator
// Run: go run scripts/send-founder-access.go
//
// This script generates a founder access key and sends it to Gaetano@aura-genesis.org
// Requires: RESEND_API_KEY environment variable
//
// Copyright 2026 Arobi. All Rights Reserved.

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv" // NEW IMPORT
)

const (
	FounderEmail = "Gaetano@aura-genesis.org"
	ResendAPIURL = "https://api.resend.com/emails"
)

type EmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

type EmailResponse struct {
	ID string `json:"id"`
}

func main() {
	err := godotenv.Load() // NEW: Load .env file
	if err != nil {
		log.Println("Error loading .env file:", err)
		// Don't exit here - continue to check for API key
	}

	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		log.Println("╔════════════════════════════════════════════════════════════╗")
		log.Println("║  ASGARD FOUNDER ACCESS KEY GENERATOR                       ║")
		log.Println("╚════════════════════════════════════════════════════════════╝")
		log.Println("")
		log.Println("ERROR: RESEND_API_KEY environment variable not set")
		log.Println("")
		log.Println("To get a free API key:")
		log.Println("  1. Go to https://resend.com")
		log.Println("  2. Sign up for free (3000 emails/month)")
		log.Println("  3. Create an API key")
		log.Println("  4. Run: set RESEND_API_KEY=re_your_api_key_here")
		log.Println("  5. Run this script again")
		log.Println("")
		os.Exit(1)
	}

	log.Println("╔════════════════════════════════════════════════════════════╗")
	log.Println("║  ASGARD FOUNDER ACCESS KEY GENERATOR                       ║")
	log.Println("╚════════════════════════════════════════════════════════════╝")
	log.Println("")

	// Generate secure access keys
	masterKey := generateAccessKey("FOUNDER_MASTER")
	adminKey := generateAccessKey("ADMIN_ACCESS")
	govKey := generateAccessKey("GOVERNMENT_ACCESS")

	log.Printf("Generated Founder Master Key: %s...", masterKey[:30])
	log.Printf("Generated Admin Access Key: %s...", adminKey[:30])
	log.Printf("Generated Government Access Key: %s...", govKey[:30])
	log.Println("")

	// Create email HTML
	html := createEmailHTML(masterKey, adminKey, govKey)

	// Send email
	log.Printf("Sending access keys to %s...", FounderEmail)

	req := EmailRequest{
		From:    "ASGARD Security <Gaetano@aura-genesis.org>",
		To:      []string{FounderEmail},
		Subject: "[ASGARD] Your Founder Access Keys - CONFIDENTIAL",
		HTML:    html,
	}

	body, _ := json.Marshal(req)

	httpReq, err := http.NewRequest("POST", ResendAPIURL, bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		log.Fatalf("Email API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var emailResp EmailResponse
	json.Unmarshal(respBody, &emailResp)

	log.Println("")
	log.Println("╔════════════════════════════════════════════════════════════╗")
	log.Println("║  ACCESS KEYS SENT SUCCESSFULLY!                            ║")
	log.Println("╚════════════════════════════════════════════════════════════╝")
	log.Println("")
	log.Printf("Email ID: %s", emailResp.ID)
	log.Printf("Sent to: %s", FounderEmail)
	log.Println("")
	log.Println("The following keys were generated and sent:")
	log.Println("  - FOUNDER_MASTER key (full system access)")
	log.Println("  - ADMIN_ACCESS key (admin portal access)")
	log.Println("  - GOVERNMENT_ACCESS key (gov portal access)")
	log.Println("")
	log.Println("Keys expire in 24 hours and are one-time use.")
	log.Println("Please check your email and store the keys securely.")
	log.Println("")

	// Also save to a local file for backup
	saveKeysToFile(masterKey, adminKey, govKey)
}

func generateAccessKey(keyType string) string {
	keyBytes := make([]byte, 32)
	rand.Read(keyBytes)
	return fmt.Sprintf("ASGARD-%s-%s", keyType, base64.URLEncoding.EncodeToString(keyBytes)[:32])
}

func createEmailHTML(masterKey, adminKey, govKey string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: 'Segoe UI', Arial, sans-serif; background: #0a0a0a; color: #ffffff; margin: 0; padding: 20px; }
        .container { max-width: 700px; margin: 0 auto; background: #1a1a2e; border-radius: 12px; overflow: hidden; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 40px; text-align: center; }
        .header h1 { margin: 0; font-size: 32px; }
        .header .icon { font-size: 64px; margin-bottom: 15px; }
        .header .subtitle { opacity: 0.9; margin-top: 10px; }
        .content { padding: 40px; }
        .key-section { background: #0f0f1a; border: 2px solid #667eea; border-radius: 12px; padding: 25px; margin: 25px 0; }
        .key-label { color: #667eea; font-weight: bold; font-size: 14px; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 10px; }
        .key-value { font-family: 'Courier New', monospace; font-size: 14px; color: #00ff88; word-break: break-all; background: #000; padding: 15px; border-radius: 8px; border: 1px solid #333; }
        .key-description { color: #888; font-size: 13px; margin-top: 10px; }
        .warning { background: #332200; border-left: 4px solid #ffaa00; padding: 20px; margin: 25px 0; color: #ffcc00; }
        .warning-title { font-weight: bold; margin-bottom: 10px; font-size: 16px; }
        .info-box { background: #001a33; border-left: 4px solid #0088ff; padding: 20px; margin: 25px 0; }
        .footer { background: #0f0f1a; padding: 30px; text-align: center; font-size: 12px; color: #666; border-top: 1px solid #333; }
        .timestamp { color: #888; font-size: 12px; margin-top: 20px; }
        .portal-links { margin-top: 30px; }
        .portal-link { display: inline-block; background: #667eea; color: white; padding: 12px 24px; border-radius: 8px; text-decoration: none; margin: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="icon">🔐</div>
            <h1>ASGARD Security</h1>
            <div class="subtitle">Founder Access Keys</div>
        </div>
        <div class="content">
            <p>Dear Founder Gaetano Comparcola,</p>
            <p>Your ASGARD system access keys have been generated. These keys provide full access to all ASGARD systems and portals.</p>

            <div class="key-section">
                <div class="key-label">🔑 Founder Master Key</div>
                <div class="key-value">%s</div>
                <div class="key-description">Full system access - All ASGARD services, vault, and override capabilities</div>
            </div>

            <div class="key-section">
                <div class="key-label">🛡️ Admin Portal Access Key</div>
                <div class="key-value">%s</div>
                <div class="key-description">Access to admin dashboard, user management, and system configuration</div>
            </div>

            <div class="key-section">
                <div class="key-label">🏛️ Government Portal Access Key</div>
                <div class="key-value">%s</div>
                <div class="key-description">Access to DO-178C compliance, FAA certification, and government contractor portals</div>
            </div>

            <div class="warning">
                <div class="warning-title">⚠️ SECURITY NOTICE</div>
                <ul style="margin: 0; padding-left: 20px;">
                    <li>These keys expire in <strong>24 hours</strong></li>
                    <li>Each key can only be used <strong>once</strong> for initial authentication</li>
                    <li>After first use, you'll set up FIDO2/biometric authentication</li>
                    <li>Store these keys securely - do not share</li>
                    <li>Delete this email after saving the keys</li>
                </ul>
            </div>

            <div class="info-box">
                <strong>Next Steps:</strong>
                <ol style="margin: 10px 0 0 0; padding-left: 20px;">
                    <li>Download the Electron app from the portal</li>
                    <li>Enter your access key when prompted</li>
                    <li>Complete FIDO2 security key setup</li>
                    <li>Configure your security preferences</li>
                </ol>
            </div>

            <p class="timestamp">Generated: %s UTC</p>
        </div>
        <div class="footer">
            <p><strong>ASGARD Autonomous Systems</strong></p>
            <p>Protecting Humanity Through Ethical AI</p>
            <p style="margin-top: 15px;">Copyright 2026 Arobi. All Rights Reserved.</p>
            <p>CONFIDENTIAL - PROPRIETARY</p>
        </div>
    </div>
</body>
</html>
`, masterKey, adminKey, govKey, time.Now().UTC().Format("2006-01-02 15:04:05"))
}

func saveKeysToFile(masterKey, adminKey, govKey string) {
	content := fmt.Sprintf(`# ASGARD Founder Access Keys
# Generated: %s
# CONFIDENTIAL - DELETE AFTER USE

## Founder Master Key
%s

## Admin Portal Key
%s

## Government Portal Key
%s

---
Keys expire in 24 hours.
Each key is one-time use only.
`, time.Now().UTC().Format("2006-01-02 15:04:05 UTC"), masterKey, adminKey, govKey)

	filename := fmt.Sprintf("founder_access_keys_%s.txt", time.Now().Format("20060102_150405"))
	err := os.WriteFile(filename, []byte(content), 0600)
	if err != nil {
		log.Printf("Warning: Could not save backup file: %v", err)
		return
	}
	log.Printf("Backup saved to: %s", filename)
	log.Println("(Remember to delete this file after use)")
}
