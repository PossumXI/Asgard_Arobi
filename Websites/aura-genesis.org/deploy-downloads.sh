#!/bin/bash

# APEX-OS-LQ Downloads Deployment Script
# This script helps set up the download infrastructure

set -e

echo "APEX-OS-LQ Downloads Deployment Script"
echo "====================================="

# Configuration
DOWNLOADS_DIR="./downloads"
PACKAGES_DIR="$DOWNLOADS_DIR/packages"
TEMP_DIR="$DOWNLOADS_DIR/temp"

# Create directory structure
echo "Creating directory structure..."
mkdir -p "$PACKAGES_DIR"
mkdir -p "$TEMP_DIR"
mkdir -p "./logs"

# Function to generate SHA256 checksum
generate_checksum() {
    local file="$1"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$file" | cut -d' ' -f1
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$file" | cut -d' ' -f1
    else
        echo "Error: No SHA256 tool found" >&2
        return 1
    fi
}

# Function to update manifest with actual checksums
update_manifest_checksums() {
    echo "Updating manifest with actual file checksums..."

    local manifest_file="$DOWNLOADS_DIR/manifest.json"

    if [ ! -f "$manifest_file" ]; then
        echo "Error: Manifest file not found at $manifest_file"
        return 1
    fi

    # Update Windows checksum
    if [ -f "$PACKAGES_DIR/apex-os-lq-windows-1.0.0.exe" ]; then
        local windows_checksum=$(generate_checksum "$PACKAGES_DIR/apex-os-lq-windows-1.0.0.exe")
        sed -i.bak "s/\"windows\": \"[^\"]*\"/\"windows\": \"$windows_checksum\"/" "$manifest_file"
    fi

    # Update macOS checksum
    if [ -f "$PACKAGES_DIR/apex-os-lq-macos-1.0.0.dmg" ]; then
        local macos_checksum=$(generate_checksum "$PACKAGES_DIR/apex-os-lq-macos-1.0.0.dmg")
        sed -i.bak "s/\"macos\": \"[^\"]*\"/\"macos\": \"$macos_checksum\"/" "$manifest_file"
    fi

    # Update Linux checksum
    if [ -f "$PACKAGES_DIR/apex-os-lq-linux-1.0.0.AppImage" ]; then
        local linux_checksum=$(generate_checksum "$PACKAGES_DIR/apex-os-lq-linux-1.0.0.AppImage")
        sed -i.bak "s/\"linux\": \"[^\"]*\"/\"linux\": \"$linux_checksum\"/" "$manifest_file"
    fi

    # Update USB creator checksum
    if [ -f "$PACKAGES_DIR/apex-os-lq-usb-creator-1.0.0.exe" ]; then
        local usb_checksum=$(generate_checksum "$PACKAGES_DIR/apex-os-lq-usb-creator-1.0.0.exe")
        sed -i.bak "s/\"usb_creator\": \"[^\"]*\"/\"usb_creator\": \"$usb_checksum\"/" "$manifest_file"
    fi

    echo "Manifest updated with actual checksums"
}

# Function to verify package integrity
verify_packages() {
    echo "Verifying package integrity..."

    local manifest_file="$DOWNLOADS_DIR/manifest.json"
    local errors=0

    # Check Windows package
    if [ -f "$PACKAGES_DIR/apex-os-lq-windows-1.0.0.exe" ]; then
        local expected=$(grep -o '"windows": "[^"]*"' "$manifest_file" | cut -d'"' -f4)
        local actual=$(generate_checksum "$PACKAGES_DIR/apex-os-lq-windows-1.0.0.exe")
        if [ "$expected" != "$actual" ]; then
            echo "ERROR: Windows package checksum mismatch!"
            echo "Expected: $expected"
            echo "Actual: $actual"
            ((errors++))
        fi
    fi

    # Similar checks for other platforms...

    if [ $errors -eq 0 ]; then
        echo "All package checksums verified successfully"
    else
        echo "ERROR: $errors checksum verification(s) failed"
        return 1
    fi
}

# Function to set permissions
set_permissions() {
    echo "Setting file permissions..."

    # Make packages readable by web server
    find "$PACKAGES_DIR" -type f -exec chmod 644 {} \;

    # Make directories traversable
    find "$DOWNLOADS_DIR" -type d -exec chmod 755 {} \;

    # Make scripts executable
    chmod +x "$0"
    if [ -f "./api/downloads.php" ]; then
        chmod 644 "./api/downloads.php"
    fi
}

# Function to create backup
create_backup() {
    echo "Creating backup of current manifest..."
    cp "$DOWNLOADS_DIR/manifest.json" "$DOWNLOADS_DIR/manifest.json.backup.$(date +%Y%m%d_%H%M%S)"
}

# Main deployment process
case "${1:-deploy}" in
    "deploy")
        echo "Starting deployment process..."

        create_backup
        update_manifest_checksums
        verify_packages
        set_permissions

        echo ""
        echo "Deployment completed successfully!"
        echo ""
        echo "Next steps:"
        echo "1. Upload the packages to your web server"
        echo "2. Update the website to show downloads (remove dev toggle)"
        echo "3. Test downloads from different platforms"
        echo "4. Monitor download analytics"
        ;;

    "verify")
        verify_packages
        ;;

    "update-checksums")
        create_backup
        update_manifest_checksums
        ;;

    "permissions")
        set_permissions
        ;;

    *)
        echo "Usage: $0 [deploy|verify|update-checksums|permissions]"
        echo ""
        echo "Commands:"
        echo "  deploy          - Full deployment process"
        echo "  verify          - Verify package integrity"
        echo "  update-checksums - Update manifest with actual checksums"
        echo "  permissions     - Set proper file permissions"
        exit 1
        ;;
esac
