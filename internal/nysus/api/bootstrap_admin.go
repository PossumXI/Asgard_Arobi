package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/base32"
	"log"
	"os"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type bootstrapAdminResult struct {
	UserID   string
	Email    string
	FullName string
	Password string
	Created  bool
}

func bootstrapAdminUser(pgDB *db.PostgresDB) bootstrapAdminResult {
	if pgDB == nil {
		return bootstrapAdminResult{}
	}
	email := strings.TrimSpace(os.Getenv("ASGARD_BOOTSTRAP_ADMIN_EMAIL"))
	password := strings.TrimSpace(os.Getenv("ASGARD_BOOTSTRAP_ADMIN_PASSWORD"))
	fullName := strings.TrimSpace(os.Getenv("ASGARD_BOOTSTRAP_ADMIN_FULL_NAME"))
	if email == "" {
		email = "Gaetano@aura-genesis.org"
	}
	if fullName == "" {
		fullName = "Gaetano Comparcola"
	}

	var existingID string
	err := pgDB.QueryRow(`SELECT id::text FROM users WHERE email = $1`, email).Scan(&existingID)
	if err == nil && existingID != "" {
		log.Printf("[Bootstrap] admin user already exists: %s", email)
		return bootstrapAdminResult{
			UserID:   existingID,
			Email:    email,
			FullName: fullName,
			Created:  false,
		}
	}
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[Bootstrap] failed to query user: %v", err)
		return bootstrapAdminResult{}
	}

	if password == "" {
		password = generateBootstrapPassword()
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[Bootstrap] failed to hash password: %v", err)
		return bootstrapAdminResult{}
	}

	userID := uuid.New()
	now := time.Now().UTC()
	_, err = pgDB.Exec(`
		INSERT INTO users (id, email, password_hash, full_name, subscription_tier, is_government, email_verified, email_verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, true, $6, $7, $8)
	`, userID, email, string(hashed), fullName, "commander", now, now, now)
	if err != nil {
		log.Printf("[Bootstrap] failed to create admin user: %v", err)
		return bootstrapAdminResult{}
	}
	log.Printf("[Bootstrap] admin user created: %s", email)
	return bootstrapAdminResult{
		UserID:   userID.String(),
		Email:    email,
		FullName: fullName,
		Password: password,
		Created:  true,
	}
}

func generateBootstrapPassword() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "Temp-Access-ChangeMe!"
	}
	encoded := strings.TrimRight(base32.StdEncoding.EncodeToString(buf), "=")
	if len(encoded) > 12 {
		encoded = encoded[:12]
	}
	return "Temp-" + encoded + "!"
}
