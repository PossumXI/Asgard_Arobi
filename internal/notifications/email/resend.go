// Package email provides email notification services using Resend API.
// Free tier: 3,000 emails/month - perfect for access key notifications.
//
// Copyright 2026 Arobi. All Rights Reserved.
package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ResendConfig holds Resend API configuration
type ResendConfig struct {
	APIKey     string
	FromEmail  string
	FromName   string
	BaseURL    string
	Timeout    time.Duration
	RetryCount int
}

// DefaultResendConfig returns default configuration
func DefaultResendConfig() ResendConfig {
	return ResendConfig{
		APIKey:     "", // Set via RESEND_API_KEY env var
		FromEmail:  "security@asgard.arobi.io",
		FromName:   "ASGARD Security",
		BaseURL:    "https://api.resend.com",
		Timeout:    30 * time.Second,
		RetryCount: 3,
	}
}

// ResendClient is the Resend email client
type ResendClient struct {
	config     ResendConfig
	httpClient *http.Client
}

// NewResendClient creates a new Resend client
func NewResendClient(cfg ResendConfig) *ResendClient {
	return &ResendClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// EmailRequest represents an email send request
type EmailRequest struct {
	From        string       `json:"from"`
	To          []string     `json:"to"`
	Subject     string       `json:"subject"`
	HTML        string       `json:"html,omitempty"`
	Text        string       `json:"text,omitempty"`
	ReplyTo     string       `json:"reply_to,omitempty"`
	BCC         []string     `json:"bcc,omitempty"`
	CC          []string     `json:"cc,omitempty"`
	Tags        []Tag        `json:"tags,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Tag represents an email tag
type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"` // Base64 encoded
	ContentType string `json:"content_type,omitempty"`
}

// EmailResponse represents the API response
type EmailResponse struct {
	ID string `json:"id"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Name       string `json:"name"`
}

// SendEmail sends an email using Resend API
func (c *ResendClient) SendEmail(ctx context.Context, req EmailRequest) (*EmailResponse, error) {
	// Set from address if not provided
	if req.From == "" {
		req.From = fmt.Sprintf("%s <%s>", c.config.FromName, c.config.FromEmail)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < c.config.RetryCount; attempt++ {
		resp, err := c.doRequest(ctx, "POST", "/emails", body)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt+1) * time.Second)
			continue
		}
		return resp, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.config.RetryCount, lastErr)
}

func (c *ResendClient) doRequest(ctx context.Context, method, path string, body []byte) (*EmailResponse, error) {
	url := c.config.BaseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		json.Unmarshal(respBody, &errResp)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, errResp.Message)
	}

	var emailResp EmailResponse
	if err := json.Unmarshal(respBody, &emailResp); err != nil {
		return nil, err
	}

	return &emailResp, nil
}

// SendAccessKeyEmail sends a new access key to the founder
func (c *ResendClient) SendAccessKeyEmail(ctx context.Context, to, accessKey, keyType string) error {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: 'Segoe UI', Arial, sans-serif; background: #0a0a0a; color: #ffffff; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: #1a1a2e; border-radius: 12px; overflow: hidden; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 28px; }
        .header .icon { font-size: 48px; margin-bottom: 10px; }
        .content { padding: 30px; }
        .key-box { background: #0f0f1a; border: 2px solid #667eea; border-radius: 8px; padding: 20px; margin: 20px 0; text-align: center; }
        .key-value { font-family: 'Courier New', monospace; font-size: 18px; color: #00ff88; word-break: break-all; letter-spacing: 1px; }
        .key-type { color: #667eea; font-weight: bold; margin-bottom: 10px; }
        .warning { background: #332200; border-left: 4px solid #ffaa00; padding: 15px; margin: 20px 0; color: #ffcc00; }
        .footer { background: #0f0f1a; padding: 20px; text-align: center; font-size: 12px; color: #666; }
        .timestamp { color: #888; margin-top: 20px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="icon">🔐</div>
            <h1>ASGARD Security</h1>
        </div>
        <div class="content">
            <p>Dear Founder,</p>
            <p>A new access key has been generated for your ASGARD systems:</p>

            <div class="key-box">
                <div class="key-type">%s</div>
                <div class="key-value">%s</div>
            </div>

            <div class="warning">
                <strong>⚠️ Security Notice:</strong><br>
                This key provides privileged access to ASGARD systems. Store it securely and never share it.
                If you did not request this key, please contact security immediately.
            </div>

            <p>This key is valid for 24 hours and can only be used once for initial authentication.</p>

            <p class="timestamp">Generated: %s UTC</p>
        </div>
        <div class="footer">
            <p>ASGARD Autonomous Systems - Protecting Humanity</p>
            <p>Copyright 2026 Arobi. All Rights Reserved.</p>
            <p>This is an automated message. Do not reply.</p>
        </div>
    </div>
</body>
</html>
`, keyType, accessKey, time.Now().UTC().Format("2006-01-02 15:04:05"))

	req := EmailRequest{
		To:      []string{to},
		Subject: fmt.Sprintf("[ASGARD] New %s Access Key Generated", keyType),
		HTML:    html,
		Tags: []Tag{
			{Name: "type", Value: "access_key"},
			{Name: "key_type", Value: keyType},
		},
	}

	_, err := c.SendEmail(ctx, req)
	return err
}

// SendSecurityAlertEmail sends a security alert to the founder
func (c *ResendClient) SendSecurityAlertEmail(ctx context.Context, to, alertType, description string, severity string) error {
	severityColor := "#00ff88" // green
	switch severity {
	case "critical":
		severityColor = "#ff0000"
	case "high":
		severityColor = "#ff6600"
	case "medium":
		severityColor = "#ffcc00"
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: 'Segoe UI', Arial, sans-serif; background: #0a0a0a; color: #ffffff; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: #1a1a2e; border-radius: 12px; overflow: hidden; }
        .header { background: %s; padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 28px; color: #000; }
        .header .icon { font-size: 48px; margin-bottom: 10px; }
        .content { padding: 30px; }
        .alert-box { background: #0f0f1a; border: 2px solid %s; border-radius: 8px; padding: 20px; margin: 20px 0; }
        .severity { display: inline-block; background: %s; color: #000; padding: 5px 15px; border-radius: 20px; font-weight: bold; }
        .footer { background: #0f0f1a; padding: 20px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="icon">🚨</div>
            <h1>Security Alert</h1>
        </div>
        <div class="content">
            <p><span class="severity">%s</span></p>

            <div class="alert-box">
                <h3>%s</h3>
                <p>%s</p>
            </div>

            <p><strong>Time:</strong> %s UTC</p>
            <p>ASGARD systems are actively monitoring and responding to this alert.</p>
        </div>
        <div class="footer">
            <p>ASGARD Security Operations Center</p>
        </div>
    </div>
</body>
</html>
`, severityColor, severityColor, severityColor, severity, alertType, description, time.Now().UTC().Format("2006-01-02 15:04:05"))

	req := EmailRequest{
		To:      []string{to},
		Subject: fmt.Sprintf("[ASGARD ALERT - %s] %s", severity, alertType),
		HTML:    html,
		Tags: []Tag{
			{Name: "type", Value: "security_alert"},
			{Name: "severity", Value: severity},
		},
	}

	_, err := c.SendEmail(ctx, req)
	return err
}

// SendVerificationEmail sends an email verification code
func (c *ResendClient) SendVerificationEmail(ctx context.Context, to, code string) error {
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <style>
        body { font-family: 'Segoe UI', Arial, sans-serif; background: #0a0a0a; color: #ffffff; margin: 0; padding: 20px; }
        .container { max-width: 600px; margin: 0 auto; background: #1a1a2e; border-radius: 12px; overflow: hidden; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 30px; text-align: center; }
        .header h1 { margin: 0; font-size: 28px; }
        .content { padding: 30px; text-align: center; }
        .code-box { background: #0f0f1a; border-radius: 8px; padding: 30px; margin: 30px 0; }
        .code { font-family: 'Courier New', monospace; font-size: 36px; color: #00ff88; letter-spacing: 8px; }
        .expiry { color: #888; margin-top: 20px; }
        .footer { background: #0f0f1a; padding: 20px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Verify Your Email</h1>
        </div>
        <div class="content">
            <p>Your verification code is:</p>
            <div class="code-box">
                <div class="code">%s</div>
            </div>
            <p class="expiry">This code expires in 10 minutes.</p>
        </div>
        <div class="footer">
            <p>ASGARD Security - Protecting Humanity</p>
        </div>
    </div>
</body>
</html>
`, code)

	req := EmailRequest{
		To:      []string{to},
		Subject: "[ASGARD] Email Verification Code",
		HTML:    html,
		Tags: []Tag{
			{Name: "type", Value: "verification"},
		},
	}

	_, err := c.SendEmail(ctx, req)
	return err
}
