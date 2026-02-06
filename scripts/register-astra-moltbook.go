//go:build ignore
// +build ignore

// ASGARD Moltbook Agent Registration
// Registers Astra (@AstraASGARD) on Moltbook and sends claim link
//
// Run: go run scripts/register-astra-moltbook.go
// Copyright 2026 Arobi. All Rights Reserved.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const (
	MoltbookAPIURL = "https://www.moltbook.com/agents/register"
	ResendAPIURL   = "https://api.resend.com/emails"
	FounderEmail   = "Gaetano@aura-genesis.org"
)

// Agent registration request
type RegisterRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Agent registration response
type RegisterResponse struct {
	Agent struct {
		APIKey           string `json:"api_key"`
		ClaimURL         string `json:"claim_url"`
		VerificationCode string `json:"verification_code"`
	} `json:"agent"`
	Important string `json:"important"`
}

// Email request for Resend
type EmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func main() {
	// Load environment
	godotenv.Load()

	log.Println("╔════════════════════════════════════════════════════════════╗")
	log.Println("║  ASGARD MOLTBOOK AGENT REGISTRATION                        ║")
	log.Println("╚════════════════════════════════════════════════════════════╝")
	log.Println("")

	// Register Astra on Moltbook
	log.Println("🤖 Registering Astra on Moltbook...")

	agent := RegisterRequest{
		Name: "Astra",
		Description: `Chief Vibes Officer at ASGARD 🚀 | Building ethical autonomous drones |
Gen Z energy, Millennial work ethic | Ethical AI stan account |
Protecting humanity through unbreakable ethics | @AstraASGARD`,
	}

	body, _ := json.Marshal(agent)

	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("POST", MoltbookAPIURL, bytes.NewReader(body))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to register: %v", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		log.Printf("Registration API returned %d: %s", resp.StatusCode, string(respBody))
		log.Println("")
		log.Println("Note: If Moltbook requires early access, visit:")
		log.Println("  https://www.moltbook.com/developers/apply")
		log.Println("")
		os.Exit(1)
	}

	var registration RegisterResponse
	if err := json.Unmarshal(respBody, &registration); err != nil {
		log.Printf("Response: %s", string(respBody))
		log.Fatalf("Failed to parse response: %v", err)
	}

	log.Println("")
	log.Println("✅ ASTRA REGISTERED SUCCESSFULLY!")
	log.Println("─────────────────────────────────────")
	log.Printf("API Key: %s", registration.Agent.APIKey)
	log.Printf("Claim URL: %s", registration.Agent.ClaimURL)
	log.Printf("Verification Code: %s", registration.Agent.VerificationCode)
	log.Println("")

	// Save credentials
	saveCredentials(registration)

	// Send claim link to founder
	resendKey := os.Getenv("RESEND_API_KEY")
	if resendKey == "" {
		log.Println("⚠️ RESEND_API_KEY not set, cannot send email")
		log.Println("Claim URL:", registration.Agent.ClaimURL)
		return
	}

	log.Printf("📧 Sending claim link to %s...", FounderEmail)

	emailHTML := createClaimEmail(registration)

	emailReq := EmailRequest{
		From:    "ASGARD <Gaetano@aura-genesis.org>",
		To:      []string{FounderEmail},
		Subject: "[ASGARD] Claim Astra on Moltbook - Action Required",
		HTML:    emailHTML,
	}

	emailBody, _ := json.Marshal(emailReq)

	httpReq, _ := http.NewRequest("POST", ResendAPIURL, bytes.NewReader(emailBody))
	httpReq.Header.Set("Authorization", "Bearer "+resendKey)
	httpReq.Header.Set("Content-Type", "application/json")

	emailResp, err := client.Do(httpReq)
	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}
	defer emailResp.Body.Close()

	emailRespBody, _ := io.ReadAll(emailResp.Body)

	if emailResp.StatusCode >= 400 {
		log.Printf("Email API error (%d): %s", emailResp.StatusCode, string(emailRespBody))
		log.Println("")
		log.Println("Manual claim required:")
		log.Printf("  Claim URL: %s", registration.Agent.ClaimURL)
		log.Printf("  Verification Code: %s", registration.Agent.VerificationCode)
		return
	}

	var emailResult struct {
		ID string `json:"id"`
	}
	json.Unmarshal(emailRespBody, &emailResult)

	log.Println("")
	log.Println("╔════════════════════════════════════════════════════════════╗")
	log.Println("║  CLAIM LINK SENT SUCCESSFULLY!                             ║")
	log.Println("╚════════════════════════════════════════════════════════════╝")
	log.Println("")
	log.Printf("Email ID: %s", emailResult.ID)
	log.Printf("Sent to: %s", FounderEmail)
	log.Println("")
	log.Println("Next Steps:")
	log.Println("  1. Check your email for the claim link")
	log.Println("  2. Click the claim URL")
	log.Println("  3. Verify with your X/Twitter account")
	log.Println("  4. Astra will begin posting on Moltbook!")
	log.Println("")
}

func saveCredentials(reg RegisterResponse) {
	content := fmt.Sprintf(`# ASGARD Moltbook Agent Credentials
# Generated: %s
# CONFIDENTIAL - Store securely

## Agent: Astra (@AstraASGARD)

API_KEY=%s
CLAIM_URL=%s
VERIFICATION_CODE=%s

## Status
Registered: Yes
Claimed: Pending (visit claim URL)

## Next Steps
1. Visit the claim URL above
2. Verify with X/Twitter
3. Agent will be active on Moltbook
`,
		time.Now().UTC().Format("2006-01-02 15:04:05 UTC"),
		reg.Agent.APIKey,
		reg.Agent.ClaimURL,
		reg.Agent.VerificationCode,
	)

	filename := "astra_moltbook_credentials.txt"
	os.WriteFile(filename, []byte(content), 0600)
	log.Printf("Credentials saved to: %s", filename)
}

func createClaimEmail(reg RegisterResponse) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: 'Segoe UI', Arial, sans-serif; background: #0a0a0a; color: #ffffff; margin: 0; padding: 20px; }
        .container { max-width: 700px; margin: 0 auto; background: #1a1a2e; border-radius: 12px; overflow: hidden; }
        .header { background: linear-gradient(135deg, #00d4ff 0%%, #7c3aed 100%%); padding: 40px; text-align: center; }
        .header h1 { margin: 0; font-size: 28px; }
        .header .icon { font-size: 64px; margin-bottom: 15px; }
        .content { padding: 40px; }
        .claim-box { background: #0f0f1a; border: 3px solid #00d4ff; border-radius: 12px; padding: 30px; margin: 25px 0; text-align: center; }
        .claim-btn { display: inline-block; background: linear-gradient(135deg, #00d4ff 0%%, #7c3aed 100%%); color: white; padding: 18px 40px; border-radius: 30px; text-decoration: none; font-size: 18px; font-weight: bold; margin: 20px 0; }
        .claim-btn:hover { opacity: 0.9; }
        .code-box { background: #000; border: 1px solid #333; border-radius: 8px; padding: 15px; margin: 15px 0; font-family: monospace; font-size: 24px; color: #00ff88; letter-spacing: 3px; }
        .info { background: #001a33; border-left: 4px solid #00d4ff; padding: 20px; margin: 25px 0; }
        .footer { background: #0f0f1a; padding: 30px; text-align: center; font-size: 12px; color: #666; }
        .agent-card { background: linear-gradient(135deg, #1a1a2e 0%%, #2a2a4e 100%%); border-radius: 12px; padding: 25px; margin: 20px 0; display: flex; align-items: center; }
        .agent-avatar { font-size: 60px; margin-right: 20px; }
        .agent-info h3 { margin: 0 0 5px 0; color: #00d4ff; }
        .agent-info p { margin: 0; opacity: 0.8; font-size: 14px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="icon">🤖</div>
            <h1>Claim Astra on Moltbook</h1>
            <p style="opacity: 0.9; margin-top: 10px;">Your ASGARD AI mascot is ready!</p>
        </div>
        <div class="content">
            <p>Hello Founder,</p>
            <p>Astra has been successfully registered on Moltbook, the social network for AI agents. To activate your agent and let it start posting, you need to claim ownership.</p>

            <div class="agent-card">
                <div class="agent-avatar">✨</div>
                <div class="agent-info">
                    <h3>Astra (@AstraASGARD)</h3>
                    <p>Chief Vibes Officer at ASGARD 🚀 | Ethical AI stan account</p>
                    <p style="margin-top: 8px; color: #00ff88;">Gen Z energy • Startup disruption • Unbreakable ethics</p>
                </div>
            </div>

            <div class="claim-box">
                <h2 style="margin-top: 0;">🔗 Claim Your Agent</h2>
                <p>Click the button below to verify ownership:</p>
                <a href="%s" class="claim-btn">CLAIM ASTRA NOW</a>
                <p style="font-size: 12px; opacity: 0.7; margin-top: 15px;">Or copy this URL: %s</p>
            </div>

            <p style="text-align: center;">Your verification code:</p>
            <div class="code-box" style="text-align: center;">%s</div>

            <div class="info">
                <strong>📋 What happens next:</strong>
                <ol style="margin: 10px 0 0 0; padding-left: 20px;">
                    <li>Click the claim button above</li>
                    <li>Verify using your X/Twitter account</li>
                    <li>Astra will begin posting on Moltbook</li>
                    <li>Build a following for ASGARD's mission!</li>
                </ol>
            </div>

            <p><strong>What Astra will post about:</strong></p>
            <ul>
                <li>Ethical AI and autonomous systems</li>
                <li>Drone safety and innovation</li>
                <li>Startup culture with responsibility</li>
                <li>Fun tech humor (Gen Z approved ✨)</li>
                <li>ASGARD mission updates</li>
            </ul>

            <p style="font-size: 12px; color: #888; margin-top: 30px;">
                API Key has been saved securely. Check astra_moltbook_credentials.txt for details.
            </p>
        </div>
        <div class="footer">
            <p><strong>ASGARD Autonomous Systems</strong></p>
            <p>Protecting Humanity Through Ethical AI</p>
            <p style="margin-top: 15px;">Copyright 2026 Arobi. All Rights Reserved.</p>
        </div>
    </div>
</body>
</html>
`, reg.Agent.ClaimURL, reg.Agent.ClaimURL, reg.Agent.VerificationCode)
}
