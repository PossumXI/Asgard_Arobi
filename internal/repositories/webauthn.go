// Package repositories provides data access for WebAuthn credentials and sessions.
package repositories

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/asgard/pandora/internal/platform/db"
	"github.com/go-webauthn/webauthn/protocol"
	webauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// WebAuthnRepository manages WebAuthn credentials and sessions.
type WebAuthnRepository struct {
	db *db.PostgresDB
}

// NewWebAuthnRepository creates a new WebAuthn repository.
func NewWebAuthnRepository(pgDB *db.PostgresDB) *WebAuthnRepository {
	return &WebAuthnRepository{db: pgDB}
}

// StoreSession persists a WebAuthn session.
func (r *WebAuthnRepository) StoreSession(userID, sessionType string, sessionData webauthn.SessionData, expiresAt time.Time) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	payload, err := json.Marshal(sessionData)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}

	query := `
		INSERT INTO webauthn_sessions (user_id, session_type, challenge, session_data, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = r.db.Exec(query, userUUID, sessionType, sessionData.Challenge, payload, expiresAt)
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}
	return nil
}

// GetLatestSession returns the most recent session for a user and type.
func (r *WebAuthnRepository) GetLatestSession(userID, sessionType string) (webauthn.SessionData, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return webauthn.SessionData{}, fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		SELECT session_data
		FROM webauthn_sessions
		WHERE user_id = $1 AND session_type = $2 AND expires_at > NOW()
		ORDER BY created_at DESC
		LIMIT 1
	`

	var data []byte
	if err := r.db.QueryRow(query, userUUID, sessionType).Scan(&data); err != nil {
		if err == sql.ErrNoRows {
			return webauthn.SessionData{}, fmt.Errorf("session not found")
		}
		return webauthn.SessionData{}, fmt.Errorf("failed to load session: %w", err)
	}

	var session webauthn.SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		return webauthn.SessionData{}, fmt.Errorf("failed to unmarshal session: %w", err)
	}
	return session, nil
}

// UpsertCredential stores or updates a credential.
func (r *WebAuthnRepository) UpsertCredential(userID string, credential *webauthn.Credential) error {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	transports := transportsToStrings(credential.Transport)
	aaguid := hex.EncodeToString(credential.Authenticator.AAGUID)

	query := `
		INSERT INTO webauthn_credentials (user_id, credential_id, public_key, attestation_type, transport, sign_count, aaguid, last_used_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (credential_id)
		DO UPDATE SET
			public_key = EXCLUDED.public_key,
			attestation_type = EXCLUDED.attestation_type,
			transport = EXCLUDED.transport,
			sign_count = EXCLUDED.sign_count,
			aaguid = EXCLUDED.aaguid,
			last_used_at = NOW()
	`

	_, err = r.db.Exec(query, userUUID, credential.ID, credential.PublicKey, credential.AttestationType, pq.Array(transports), credential.Authenticator.SignCount, aaguid)
	if err != nil {
		return fmt.Errorf("failed to upsert credential: %w", err)
	}
	return nil
}

// UpdateCredential updates the sign count and last used timestamp.
func (r *WebAuthnRepository) UpdateCredential(userID string, credential *webauthn.Credential) error {
	return r.UpsertCredential(userID, credential)
}

// GetCredentialsByUserID retrieves credentials for a user.
func (r *WebAuthnRepository) GetCredentialsByUserID(userID string) ([]webauthn.Credential, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	query := `
		SELECT credential_id, public_key, attestation_type, transport, sign_count, aaguid
		FROM webauthn_credentials
		WHERE user_id = $1
	`

	rows, err := r.db.Query(query, userUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to query credentials: %w", err)
	}
	defer rows.Close()

	credentials := make([]webauthn.Credential, 0)
	for rows.Next() {
		var (
			credID         []byte
			publicKey      []byte
			attestation    string
			transportArray []string
			signCount      uint32
			aaguidStr      string
		)
		if err := rows.Scan(&credID, &publicKey, &attestation, pq.Array(&transportArray), &signCount, &aaguidStr); err != nil {
			return nil, fmt.Errorf("failed to scan credential: %w", err)
		}

		aaguidBytes, _ := hex.DecodeString(aaguidStr)
		credential := webauthn.Credential{
			ID:              credID,
			PublicKey:       publicKey,
			AttestationType: attestation,
			Transport:       stringsToTransports(transportArray),
			Authenticator: webauthn.Authenticator{
				AAGUID:    aaguidBytes,
				SignCount: signCount,
			},
		}
		credentials = append(credentials, credential)
	}

	return credentials, nil
}

func transportsToStrings(transports []protocol.AuthenticatorTransport) []string {
	result := make([]string, 0, len(transports))
	for _, t := range transports {
		result = append(result, string(t))
	}
	return result
}

func stringsToTransports(values []string) []protocol.AuthenticatorTransport {
	result := make([]protocol.AuthenticatorTransport, 0, len(values))
	for _, v := range values {
		result = append(result, protocol.AuthenticatorTransport(v))
	}
	return result
}
