# Founder Device Registry

## Purpose

This directory stores registration information for devices associated with protected persons. This enables ASGARD systems to:

- Verify device authenticity
- Provide secure communication channels
- Enable location-based protection services (when authorized)
- Coordinate emergency response

## Registration Format

Devices should be registered in `registry.yaml` with the following format:

```yaml
devices:
  - device_id: "unique-device-identifier"
    owner_id: "ASGARD-001"  # Protected person ID
    device_type: "smartphone|tablet|laptop|wearable|tracker"
    device_name: "Friendly name for device"
    registration_date: "2026-02-05"
    capabilities:
      - gps_location
      - secure_messaging
      - emergency_alert
      - biometric_auth
    network_identifiers:
      wifi_mac: "XX:XX:XX:XX:XX:XX"  # Optional
      bluetooth_mac: "XX:XX:XX:XX:XX:XX"  # Optional
      imei: "XXXXXXXXXXXXXXX"  # Optional, for cellular
    security:
      encryption_enabled: true
      remote_wipe_capable: true
      fido2_registered: true
    status: "active|inactive|lost|retired"
```

## Security Requirements

- All device data is encrypted at rest
- Access requires Tier 7 clearance or Founder authorization
- All access is logged and audited
- Device credentials are stored in the Security Vault

## Adding Devices

1. Verify device ownership with protected person
2. Register device identifiers securely
3. Configure ASGARD secure communication app
4. Enable location services (optional, with consent)
5. Test emergency alert functionality

## Emergency Protocols

If a registered device triggers an alert:
1. GIRU Security is immediately notified
2. Device location is secured (if available)
3. All ASGARD systems enter protection mode
4. Founder is notified through backup channels

---

**Classification:** MAXIMUM SECURITY
**Access:** Tier 7+ or Founder Authorization
