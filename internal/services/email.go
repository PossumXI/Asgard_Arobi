package services

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// EmailService handles email sending for the application.
type EmailService struct {
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	fromEmail    string
	fromName     string
}

// NewEmailService creates a new email service.
func NewEmailService() *EmailService {
	return &EmailService{
		smtpHost:     getEnvOrDefaultShared("SMTP_HOST", "smtp.gmail.com"),
		smtpPort:     getEnvOrDefaultShared("SMTP_PORT", "587"),
		smtpUser:     os.Getenv("SMTP_USER"),
		smtpPassword: os.Getenv("SMTP_PASSWORD"),
		fromEmail:    getEnvOrDefaultShared("SMTP_FROM_EMAIL", "Gaetano@aura-genesis.org"),
		fromName:     getEnvOrDefaultShared("SMTP_FROM_NAME", "ASGARD"),
	}
}

// ErrSMTPNotConfigured is returned when SMTP credentials are missing in production.
var ErrSMTPNotConfigured = fmt.Errorf("SMTP credentials not configured: SMTP_USER and SMTP_PASSWORD environment variables are required for production email delivery")

// SendEmail sends an email.
func (es *EmailService) SendEmail(to, subject, body string) error {
	if es.smtpUser == "" || es.smtpPassword == "" {
		// Only allow console fallback in development mode
		env := os.Getenv("ASGARD_ENV")
		if env == "development" {
			fmt.Printf("[EMAIL-DEV] To: %s, Subject: %s\n%s\n", to, subject, body)
			return nil
		}
		// In production, missing SMTP credentials is an error
		return ErrSMTPNotConfigured
	}

	auth := smtp.PlainAuth("", es.smtpUser, es.smtpPassword, es.smtpHost)

	msg := []byte(fmt.Sprintf("From: %s <%s>\r\n", es.fromName, es.fromEmail) +
		fmt.Sprintf("To: %s\r\n", to) +
		fmt.Sprintf("Subject: %s\r\n", subject) +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	addr := fmt.Sprintf("%s:%s", es.smtpHost, es.smtpPort)
	return smtp.SendMail(addr, auth, es.fromEmail, []string{to}, msg)
}

// SendPasswordResetEmail sends a password reset email.
func (es *EmailService) SendPasswordResetEmail(to, resetToken string) error {
	resetURL := fmt.Sprintf("%s/auth/reset-password?token=%s",
		getEnvOrDefaultShared("FRONTEND_URL", "http://localhost:5173"), resetToken)

	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Password Reset Request</h2>
			<p>You requested to reset your password for your ASGARD account.</p>
			<p>Click the link below to reset your password:</p>
			<p><a href="%s">Reset Password</a></p>
			<p>This link will expire in 1 hour.</p>
			<p>If you did not request this, please ignore this email.</p>
		</body>
		</html>
	`, resetURL)

	return es.SendEmail(to, "ASGARD Password Reset", body)
}

// SendEmailVerification sends an email verification email.
func (es *EmailService) SendEmailVerification(to, verificationToken string) error {
	verifyURL := fmt.Sprintf("%s/auth/verify-email?token=%s",
		getEnvOrDefaultShared("FRONTEND_URL", "http://localhost:5173"), verificationToken)

	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Verify Your Email Address</h2>
			<p>Thank you for signing up for ASGARD.</p>
			<p>Please verify your email address by clicking the link below:</p>
			<p><a href="%s">Verify Email</a></p>
			<p>This link will expire in 24 hours.</p>
		</body>
		</html>
	`, verifyURL)

	return es.SendEmail(to, "Verify Your ASGARD Email", body)
}

// SendGovernmentNotification sends a notification to government users.
func (es *EmailService) SendGovernmentNotification(to, subject, message string) error {
	tmpl := `
		<html>
		<body>
			<h2>ASGARD Government Portal Notification</h2>
			<h3>{{.Subject}}</h3>
			<div>{{.Message}}</div>
			<hr>
			<p><small>This is an automated message from the ASGARD Government Portal.</small></p>
		</body>
		</html>
	`

	t, err := template.New("gov_notification").Parse(tmpl)
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	err = t.Execute(&buf, map[string]string{
		"Subject": subject,
		"Message": message,
	})
	if err != nil {
		return err
	}

	return es.SendEmail(to, fmt.Sprintf("[ASGARD Gov] %s", subject), buf.String())
}

// SendAccessCodeEmail sends a clearance access code email.
func (es *EmailService) SendAccessCodeEmail(to, accessCode string, expiresAt time.Time, scope, clearance string) error {
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>ASGARD Access Code</h2>
			<p>Your access code is:</p>
			<p><strong>%s</strong></p>
			<p>Scope: %s</p>
			<p>Clearance: %s</p>
			<p>Expires at: %s (UTC)</p>
			<p>If you did not request this code, contact security immediately.</p>
		</body>
		</html>
	`, accessCode, scope, clearance, expiresAt.UTC().Format(time.RFC3339))

	return es.SendEmail(to, "ASGARD Access Code", body)
}

// SendSubscriptionConfirmation sends a subscription confirmation email.
func (es *EmailService) SendSubscriptionConfirmation(to, tier string) error {
	body := fmt.Sprintf(`
		<html>
		<body>
			<h2>Welcome to ASGARD %s Tier</h2>
			<p>Thank you for subscribing to ASGARD %s tier!</p>
			<p>You now have access to:</p>
			<ul>
				<li>24/7 real-time streaming feeds</li>
				<li>Priority alerts and notifications</li>
				<li>Mission tracking and updates</li>
			</ul>
			<p>Access your dashboard at: <a href="%s/dashboard">Dashboard</a></p>
		</body>
		</html>
	`, strings.Title(tier), strings.Title(tier), getEnvOrDefaultShared("FRONTEND_URL", "http://localhost:5173"))

	return es.SendEmail(to, fmt.Sprintf("Welcome to ASGARD %s", strings.Title(tier)), body)
}
