package api

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func bootstrapAdminUser(pgDB *db.PostgresDB) {
	if pgDB == nil {
		return
	}
	email := strings.TrimSpace(os.Getenv("ASGARD_BOOTSTRAP_ADMIN_EMAIL"))
	password := strings.TrimSpace(os.Getenv("ASGARD_BOOTSTRAP_ADMIN_PASSWORD"))
	fullName := strings.TrimSpace(os.Getenv("ASGARD_BOOTSTRAP_ADMIN_FULL_NAME"))
	if email == "" || password == "" {
		return
	}
	if fullName == "" {
		fullName = "ASGARD Administrator"
	}

	var existingID string
	err := pgDB.QueryRow(`SELECT id::text FROM users WHERE email = $1`, email).Scan(&existingID)
	if err == nil && existingID != "" {
		log.Printf("[Bootstrap] admin user already exists: %s", email)
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[Bootstrap] failed to hash password: %v", err)
		return
	}

	now := time.Now().UTC()
	_, err = pgDB.Exec(`
		INSERT INTO users (id, email, password_hash, full_name, subscription_tier, is_government, email_verified, email_verified_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, true, $6, $7, $8)
	`, uuid.New(), email, string(hashed), fullName, "commander", now, now, now)
	if err != nil {
		log.Printf("[Bootstrap] failed to create admin user: %v", err)
		return
	}
	log.Printf("[Bootstrap] admin user created: %s", email)
}
