package api

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/asgard/pandora/internal/services"
)

func bootstrapAccessCode(
	accessCodeService *services.AccessCodeService,
	admin bootstrapAdminResult,
) {
	if accessCodeService == nil || admin.UserID == "" {
		return
	}

	rotateOnStart := strings.TrimSpace(os.Getenv("ASGARD_BOOTSTRAP_ROTATE_ON_START"))
	if strings.EqualFold(rotateOnStart, "true") || rotateOnStart == "1" {
		result, err := accessCodeService.RotateForUser(context.Background(), admin.UserID, admin.UserID)
		if err != nil {
			return
		}
		writeBootstrapDoc(admin, result.Code, result.Record.ExpiresAt)
		return
	}

	result, err := accessCodeService.IssueForUser(context.Background(), services.AccessCodeIssueRequest{
		UserID:                admin.UserID,
		CreatedBy:             admin.UserID,
		ClearanceLevel:         "government",
		Scope:                 "all",
		RotationIntervalHours: 24,
	})
	if err != nil {
		return
	}
	writeBootstrapDoc(admin, result.Code, result.Record.ExpiresAt)
}

func writeBootstrapDoc(admin bootstrapAdminResult, accessCode string, expiresAt time.Time) {
	docPath := strings.TrimSpace(os.Getenv("ASGARD_BOOTSTRAP_OUTPUT_PATH"))
	if docPath == "" {
		docPath = filepath.Join("Documentation", "Bootstrap_Access.md")
	}
	root := strings.TrimSpace(os.Getenv("ASGARD_ROOT"))
	if root != "" {
		docPath = filepath.Join(root, docPath)
	}

	content := strings.Builder{}
	content.WriteString("# ASGARD Bootstrap Access\n\n")
	content.WriteString("Generated: " + time.Now().UTC().Format(time.RFC3339) + " UTC\n\n")
	content.WriteString("## Admin Profile\n")
	content.WriteString("- Name: " + admin.FullName + "\n")
	content.WriteString("- Email: " + admin.Email + "\n")
	content.WriteString("- User ID: " + admin.UserID + "\n")
	if admin.Created {
		content.WriteString("- Status: Created on boot\n")
	} else {
		content.WriteString("- Status: Existing user\n")
	}
	if admin.Password != "" {
		content.WriteString("- Temporary Password: " + admin.Password + "\n")
	} else {
		content.WriteString("- Temporary Password: (unchanged)\n")
	}
	content.WriteString("\n## Access Code\n")
	content.WriteString("- Code: " + accessCode + "\n")
	content.WriteString("- Scope: all\n")
	content.WriteString("- Clearance: government\n")
	content.WriteString("- Expires At: " + expiresAt.UTC().Format(time.RFC3339) + " UTC\n")

	_ = os.MkdirAll(filepath.Dir(docPath), 0o755)
	_ = os.WriteFile(docPath, []byte(content.String()), 0o600)
}
