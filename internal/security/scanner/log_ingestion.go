package scanner

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Compiled regex patterns for log parsing
var (
	// IPv4 address pattern
	ipv4Pattern = regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})\b`)
	// IPv6 address pattern (simplified)
	ipv6Pattern = regexp.MustCompile(`\b([0-9a-fA-F:]{2,}::[0-9a-fA-F:]*|[0-9a-fA-F]{1,4}(:[0-9a-fA-F]{1,4}){7})\b`)
	// Port number pattern
	portPattern = regexp.MustCompile(`\bport[:\s]+(\d{1,5})\b|\b:(\d{1,5})\b`)
	// RFC 5424 syslog pattern: <PRI>VERSION TIMESTAMP HOSTNAME APP-NAME PROCID MSGID MSG
	rfc5424Pattern = regexp.MustCompile(`^<(\d{1,3})>(\d+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s+(\S+)\s*(.*)$`)
	// RFC 3164 syslog pattern: <PRI>TIMESTAMP HOSTNAME TAG: MSG
	rfc3164Pattern = regexp.MustCompile(`^<(\d{1,3})>(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})\s+(\S+)\s+(\S+?)(?:\[(\d+)\])?:\s*(.*)$`)
	// Apache/Nginx Combined Log Format pattern
	webLogPattern = regexp.MustCompile(`^(\S+)\s+\S+\s+\S+\s+\[([^\]]+)\]\s+"(\S+)\s+(\S+)\s+(\S+)"\s+(\d{3})\s+(\d+|-)\s+"([^"]*)"\s+"([^"]*)"`)
	// Common timestamp patterns
	timestampPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?)`), // ISO 8601
		regexp.MustCompile(`(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2})`),                                // YYYY-MM-DD HH:MM:SS
		regexp.MustCompile(`(\d{2}/\w{3}/\d{4}:\d{2}:\d{2}:\d{2}\s+[+-]\d{4})`),                      // Apache format
		regexp.MustCompile(`(\w{3}\s+\d{1,2}\s+\d{2}:\d{2}:\d{2})`),                                  // Syslog format
	}
)

// LogIngestionScanner reads and analyzes log files for security events.
type LogIngestionScanner struct {
	analyzer   *TrafficAnalyzer
	stats      Statistics
	mu         sync.RWMutex
	logSources []*LogSource
	running    bool
	onAnomaly  func(*Anomaly)
}

// LogSource represents a log file or stream to monitor.
type LogSource struct {
	Path     string
	File     *os.File
	Reader   *bufio.Reader
	Type     string // "syslog", "apache", "nginx", "json"
	LastRead time.Time
}

// NewLogIngestionScanner creates a new log ingestion scanner.
func NewLogIngestionScanner() *LogIngestionScanner {
	return &LogIngestionScanner{
		analyzer:   NewTrafficAnalyzer(),
		logSources: make([]*LogSource, 0),
		stats: Statistics{
			StartTime: time.Now(),
		},
	}
}

// AddLogSource adds a log file to monitor.
func (lis *LogIngestionScanner) AddLogSource(path, logType string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", path, err)
	}

	source := &LogSource{
		Path:     path,
		File:     file,
		Reader:   bufio.NewReader(file),
		Type:     logType,
		LastRead: time.Now(),
	}

	lis.mu.Lock()
	lis.logSources = append(lis.logSources, source)
	lis.mu.Unlock()

	return nil
}

// Start begins log ingestion and analysis.
func (lis *LogIngestionScanner) Start(ctx context.Context) error {
	lis.mu.Lock()
	if lis.running {
		lis.mu.Unlock()
		return nil
	}
	lis.running = true
	lis.mu.Unlock()

	// Start processing each log source
	for _, source := range lis.logSources {
		go lis.processLogSource(ctx, source)
	}

	return nil
}

// Stop stops log ingestion.
func (lis *LogIngestionScanner) Stop() error {
	lis.mu.Lock()
	defer lis.mu.Unlock()
	lis.running = false

	for _, source := range lis.logSources {
		if source.File != nil {
			source.File.Close()
		}
	}

	return nil
}

// ScanPacket implements Scanner interface (for compatibility).
func (lis *LogIngestionScanner) ScanPacket(ctx context.Context, packet PacketInfo) (*Anomaly, error) {
	lis.mu.Lock()
	lis.stats.PacketsScanned++
	lis.mu.Unlock()

	return lis.analyzer.AnalyzePacket(ctx, packet)
}

// GetStatistics returns scanner statistics.
func (lis *LogIngestionScanner) GetStatistics() Statistics {
	lis.mu.RLock()
	defer lis.mu.RUnlock()
	return lis.stats
}

// SetAnomalyCallback sets a callback for detected anomalies.
func (lis *LogIngestionScanner) SetAnomalyCallback(callback func(*Anomaly)) {
	lis.mu.Lock()
	defer lis.mu.Unlock()
	lis.onAnomaly = callback
}

// processLogSource processes a single log source.
func (lis *LogIngestionScanner) processLogSource(ctx context.Context, source *LogSource) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			line, err := source.Reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					// Wait for new log entries
					time.Sleep(1 * time.Second)
					continue
				}
				return
			}

			// Parse log line based on type
			packetInfo := lis.parseLogLine(line, source.Type)
			if packetInfo != nil {
				anomaly, _ := lis.analyzer.AnalyzePacket(ctx, *packetInfo)
				if anomaly != nil {
					lis.mu.Lock()
					lis.stats.AnomaliesDetected++
					if anomaly.Severity == ThreatLevelCritical || anomaly.Severity == ThreatLevelHigh {
						lis.stats.ThreatsBlocked++
					}
					lis.mu.Unlock()
					if lis.onAnomaly != nil {
						lis.onAnomaly(anomaly)
					}
				}
			}

			source.LastRead = time.Now()
		}
	}
}

// parseLogLine parses a log line into PacketInfo.
func (lis *LogIngestionScanner) parseLogLine(line, logType string) *PacketInfo {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	switch logType {
	case "syslog":
		return lis.parseSyslog(line)
	case "apache", "nginx":
		return lis.parseWebLog(line)
	case "json":
		return lis.parseJSONLog(line)
	default:
		return lis.parseGenericLog(line)
	}
}

// SyslogParsedData holds extracted syslog fields
type SyslogParsedData struct {
	Priority   int
	Facility   int
	Severity   int
	Version    int
	Timestamp  time.Time
	Hostname   string
	AppName    string
	ProcID     string
	MsgID      string
	Message    string
	SourceIP   net.IP
	SourcePort int
}

// parseSyslog parses RFC 5424/3164 syslog format.
// RFC 5424: <PRI>VERSION TIMESTAMP HOSTNAME APP-NAME PROCID MSGID MSG
// RFC 3164: <PRI>TIMESTAMP HOSTNAME TAG[PID]: MSG
func (lis *LogIngestionScanner) parseSyslog(line string) *PacketInfo {
	var parsed SyslogParsedData
	parsed.Timestamp = time.Now() // Default to current time

	// Try RFC 5424 format first
	if matches := rfc5424Pattern.FindStringSubmatch(line); len(matches) > 0 {
		parsed = lis.parseRFC5424(matches)
	} else if matches := rfc3164Pattern.FindStringSubmatch(line); len(matches) > 0 {
		// Try RFC 3164 format
		parsed = lis.parseRFC3164(matches)
	} else {
		// Fallback: try to extract basic info
		return lis.parseSyslogFallback(line)
	}

	// Extract source IP from message content if not already found
	if parsed.SourceIP == nil {
		if ips := ipv4Pattern.FindStringSubmatch(parsed.Message); len(ips) > 1 {
			parsed.SourceIP = net.ParseIP(ips[1])
		}
	}

	// Extract port from message content if not already found
	if parsed.SourcePort == 0 {
		if ports := portPattern.FindStringSubmatch(parsed.Message); len(ports) > 1 {
			for _, p := range ports[1:] {
				if p != "" {
					if port, err := strconv.Atoi(p); err == nil && port > 0 && port <= 65535 {
						parsed.SourcePort = port
						break
					}
				}
			}
		}
	}

	// Determine protocol based on message content
	protocol := "SYSLOG"
	if strings.Contains(strings.ToLower(parsed.Message), "ssh") {
		protocol = "SSH"
	} else if strings.Contains(strings.ToLower(parsed.Message), "http") {
		protocol = "HTTP"
	} else if strings.Contains(strings.ToLower(parsed.Message), "ftp") {
		protocol = "FTP"
	}

	return &PacketInfo{
		SourceIP:   parsed.SourceIP,
		SourcePort: parsed.SourcePort,
		Protocol:   protocol,
		Size:       len(line),
		Timestamp:  parsed.Timestamp,
		Payload:    []byte(parsed.Message),
		Flags:      fmt.Sprintf("facility=%d,severity=%d", parsed.Facility, parsed.Severity),
	}
}

// parseRFC5424 parses RFC 5424 syslog format
func (lis *LogIngestionScanner) parseRFC5424(matches []string) SyslogParsedData {
	var parsed SyslogParsedData

	// Extract PRI value and calculate facility/severity
	if pri, err := strconv.Atoi(matches[1]); err == nil {
		parsed.Priority = pri
		parsed.Facility = pri / 8
		parsed.Severity = pri % 8
	}

	// Version
	if ver, err := strconv.Atoi(matches[2]); err == nil {
		parsed.Version = ver
	}

	// Timestamp (ISO 8601 format)
	if ts, err := time.Parse(time.RFC3339, matches[3]); err == nil {
		parsed.Timestamp = ts
	} else if ts, err := time.Parse("2006-01-02T15:04:05.000000Z", matches[3]); err == nil {
		parsed.Timestamp = ts
	} else if ts, err := time.Parse("2006-01-02T15:04:05Z", matches[3]); err == nil {
		parsed.Timestamp = ts
	} else {
		parsed.Timestamp = time.Now()
	}

	parsed.Hostname = matches[4]
	parsed.AppName = matches[5]
	parsed.ProcID = matches[6]
	parsed.MsgID = matches[7]
	parsed.Message = matches[8]

	// Try to parse hostname as IP
	if ip := net.ParseIP(parsed.Hostname); ip != nil {
		parsed.SourceIP = ip
	}

	return parsed
}

// parseRFC3164 parses RFC 3164 syslog format
func (lis *LogIngestionScanner) parseRFC3164(matches []string) SyslogParsedData {
	var parsed SyslogParsedData

	// Extract PRI value and calculate facility/severity
	if pri, err := strconv.Atoi(matches[1]); err == nil {
		parsed.Priority = pri
		parsed.Facility = pri / 8
		parsed.Severity = pri % 8
	}

	// Timestamp (BSD syslog format: "Jan  2 15:04:05")
	// Add current year since RFC 3164 doesn't include year
	tsStr := matches[2]
	currentYear := time.Now().Year()
	if ts, err := time.Parse("Jan 2 15:04:05", tsStr); err == nil {
		parsed.Timestamp = ts.AddDate(currentYear, 0, 0)
	} else if ts, err := time.Parse("Jan  2 15:04:05", tsStr); err == nil {
		parsed.Timestamp = ts.AddDate(currentYear, 0, 0)
	} else {
		parsed.Timestamp = time.Now()
	}

	parsed.Hostname = matches[3]
	parsed.AppName = matches[4]
	if len(matches) > 5 && matches[5] != "" {
		parsed.ProcID = matches[5]
	}
	parsed.Message = matches[6]

	// Try to parse hostname as IP
	if ip := net.ParseIP(parsed.Hostname); ip != nil {
		parsed.SourceIP = ip
	}

	return parsed
}

// parseSyslogFallback handles non-standard syslog formats
func (lis *LogIngestionScanner) parseSyslogFallback(line string) *PacketInfo {
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return nil
	}

	var sourceIP net.IP
	var sourcePort int

	// Extract IP addresses
	if ips := ipv4Pattern.FindStringSubmatch(line); len(ips) > 1 {
		sourceIP = net.ParseIP(ips[1])
	}

	// Extract ports
	if ports := portPattern.FindStringSubmatch(line); len(ports) > 1 {
		for _, p := range ports[1:] {
			if p != "" {
				if port, err := strconv.Atoi(p); err == nil && port > 0 && port <= 65535 {
					sourcePort = port
					break
				}
			}
		}
	}

	// Try to extract timestamp
	timestamp := time.Now()
	for _, pattern := range timestampPatterns {
		if matches := pattern.FindStringSubmatch(line); len(matches) > 1 {
			if ts := lis.parseTimestamp(matches[1]); !ts.IsZero() {
				timestamp = ts
				break
			}
		}
	}

	return &PacketInfo{
		SourceIP:   sourceIP,
		SourcePort: sourcePort,
		Protocol:   "SYSLOG",
		Size:       len(line),
		Timestamp:  timestamp,
		Payload:    []byte(line),
	}
}

// WebLogParsedData holds extracted web log fields
type WebLogParsedData struct {
	SourceIP   net.IP
	Timestamp  time.Time
	Method     string
	Path       string
	Protocol   string
	StatusCode int
	Size       int
	Referer    string
	UserAgent  string
}

// parseWebLog parses Apache/Nginx Combined Log Format.
// Format: IP - - [timestamp] "METHOD PATH HTTP/1.1" STATUS SIZE "referer" "user-agent"
func (lis *LogIngestionScanner) parseWebLog(line string) *PacketInfo {
	matches := webLogPattern.FindStringSubmatch(line)
	if len(matches) < 10 {
		// Try fallback parsing
		return lis.parseWebLogFallback(line)
	}

	var parsed WebLogParsedData

	// Source IP (first capture group)
	parsed.SourceIP = net.ParseIP(matches[1])

	// Timestamp (format: "02/Jan/2006:15:04:05 -0700")
	if ts, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[2]); err == nil {
		parsed.Timestamp = ts
	} else {
		parsed.Timestamp = time.Now()
	}

	// HTTP Method
	parsed.Method = matches[3]

	// Request path
	parsed.Path = matches[4]

	// HTTP Protocol version
	parsed.Protocol = matches[5]

	// Status code
	if status, err := strconv.Atoi(matches[6]); err == nil {
		parsed.StatusCode = status
	}

	// Response size
	if matches[7] != "-" {
		if size, err := strconv.Atoi(matches[7]); err == nil {
			parsed.Size = size
		}
	}

	// Referer
	parsed.Referer = matches[8]

	// User-Agent
	parsed.UserAgent = matches[9]

	// Determine destination port based on protocol
	destPort := 80
	if strings.Contains(strings.ToLower(parsed.Protocol), "https") || strings.Contains(line, ":443") {
		destPort = 443
	}

	// Build flags with useful metadata
	flags := fmt.Sprintf("method=%s,status=%d", parsed.Method, parsed.StatusCode)

	// Detect potential threats based on patterns
	payload := []byte(parsed.Path + " " + parsed.UserAgent)
	if lis.detectSuspiciousWebRequest(parsed) {
		flags += ",suspicious=true"
	}

	return &PacketInfo{
		SourceIP:  parsed.SourceIP,
		DestPort:  destPort,
		Protocol:  "HTTP",
		Size:      parsed.Size,
		Timestamp: parsed.Timestamp,
		Payload:   payload,
		Flags:     flags,
	}
}

// parseWebLogFallback handles non-standard web log formats
func (lis *LogIngestionScanner) parseWebLogFallback(line string) *PacketInfo {
	parts := strings.Fields(line)
	if len(parts) < 4 {
		return nil
	}

	var sourceIP net.IP
	var statusCode int
	var method string
	timestamp := time.Now()

	// First field is usually the IP
	sourceIP = net.ParseIP(parts[0])

	// Look for HTTP methods
	httpMethods := []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH", "CONNECT", "TRACE"}
	for _, field := range parts {
		cleanField := strings.Trim(field, "\"")
		for _, m := range httpMethods {
			if cleanField == m {
				method = m
				break
			}
		}
	}

	// Look for status codes (3-digit numbers in typical range)
	for _, field := range parts {
		if code, err := strconv.Atoi(field); err == nil && code >= 100 && code < 600 {
			statusCode = code
			break
		}
	}

	// Try to extract timestamp from bracketed section
	bracketStart := strings.Index(line, "[")
	bracketEnd := strings.Index(line, "]")
	if bracketStart != -1 && bracketEnd > bracketStart {
		tsStr := line[bracketStart+1 : bracketEnd]
		if ts, err := time.Parse("02/Jan/2006:15:04:05 -0700", tsStr); err == nil {
			timestamp = ts
		}
	}

	return &PacketInfo{
		SourceIP:  sourceIP,
		DestPort:  80,
		Protocol:  "HTTP",
		Size:      len(line),
		Timestamp: timestamp,
		Payload:   []byte(line),
		Flags:     fmt.Sprintf("method=%s,status=%d", method, statusCode),
	}
}

// detectSuspiciousWebRequest checks for common web attack patterns
func (lis *LogIngestionScanner) detectSuspiciousWebRequest(parsed WebLogParsedData) bool {
	path := strings.ToLower(parsed.Path)
	userAgent := strings.ToLower(parsed.UserAgent)

	// SQL injection patterns
	sqlPatterns := []string{"'--", "union select", "1=1", "' or ", "drop table", "insert into"}
	for _, pattern := range sqlPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	// Directory traversal
	if strings.Contains(path, "../") || strings.Contains(path, "..\\") {
		return true
	}

	// Command injection
	cmdPatterns := []string{";", "|", "`", "$("}
	for _, pattern := range cmdPatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	// Suspicious user agents
	suspiciousAgents := []string{"sqlmap", "nikto", "nmap", "masscan", "dirbuster", "gobuster"}
	for _, agent := range suspiciousAgents {
		if strings.Contains(userAgent, agent) {
			return true
		}
	}

	// High error status codes from single IP might indicate scanning
	if parsed.StatusCode >= 400 {
		return true
	}

	return false
}

// JSONLogParsedData holds extracted JSON log fields
type JSONLogParsedData struct {
	Timestamp  time.Time
	Level      string
	Message    string
	SourceIP   net.IP
	DestIP     net.IP
	SourcePort int
	DestPort   int
	Protocol   string
	Extra      map[string]interface{}
}

// parseJSONLog parses JSON structured logs.
// Handles common JSON log formats and extracts relevant security fields.
func (lis *LogIngestionScanner) parseJSONLog(line string) *PacketInfo {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(line), &data); err != nil {
		// Not valid JSON, try to parse as generic
		return lis.parseGenericLog(line)
	}

	parsed := lis.extractJSONFields(data)

	// Determine protocol from log data
	protocol := parsed.Protocol
	if protocol == "" {
		protocol = "JSON"
	}

	// Build payload from message and extra data
	payload := []byte(parsed.Message)

	// Build flags from log level and extra data
	flags := ""
	if parsed.Level != "" {
		flags = "level=" + parsed.Level
	}

	return &PacketInfo{
		SourceIP:   parsed.SourceIP,
		DestIP:     parsed.DestIP,
		SourcePort: parsed.SourcePort,
		DestPort:   parsed.DestPort,
		Protocol:   protocol,
		Size:       len(line),
		Timestamp:  parsed.Timestamp,
		Payload:    payload,
		Flags:      flags,
	}
}

// extractJSONFields extracts common fields from JSON log data
func (lis *LogIngestionScanner) extractJSONFields(data map[string]interface{}) JSONLogParsedData {
	var parsed JSONLogParsedData
	parsed.Timestamp = time.Now()
	parsed.Extra = make(map[string]interface{})

	// Common timestamp field names
	timestampFields := []string{"timestamp", "time", "@timestamp", "ts", "datetime", "date", "created_at"}
	for _, field := range timestampFields {
		if val, ok := data[field]; ok {
			if ts := lis.parseTimestampValue(val); !ts.IsZero() {
				parsed.Timestamp = ts
				break
			}
		}
	}

	// Common level/severity field names
	levelFields := []string{"level", "severity", "log_level", "loglevel", "priority"}
	for _, field := range levelFields {
		if val, ok := data[field].(string); ok {
			parsed.Level = val
			break
		}
	}

	// Common message field names
	messageFields := []string{"message", "msg", "log", "text", "body", "content"}
	for _, field := range messageFields {
		if val, ok := data[field].(string); ok {
			parsed.Message = val
			break
		}
	}

	// Source IP fields
	sourceIPFields := []string{"source_ip", "src_ip", "sourceip", "srcip", "client_ip", "clientip", "ip", "remote_addr", "remote_ip"}
	for _, field := range sourceIPFields {
		if ip := lis.extractIPFromValue(data, field); ip != nil {
			parsed.SourceIP = ip
			break
		}
	}

	// Destination IP fields
	destIPFields := []string{"dest_ip", "dst_ip", "destip", "dstip", "server_ip", "serverip", "target_ip", "destination_ip"}
	for _, field := range destIPFields {
		if ip := lis.extractIPFromValue(data, field); ip != nil {
			parsed.DestIP = ip
			break
		}
	}

	// Source port fields
	sourcePortFields := []string{"source_port", "src_port", "sourceport", "srcport", "client_port", "sport"}
	for _, field := range sourcePortFields {
		if port := lis.extractPortFromValue(data, field); port > 0 {
			parsed.SourcePort = port
			break
		}
	}

	// Destination port fields
	destPortFields := []string{"dest_port", "dst_port", "destport", "dstport", "server_port", "dport", "port"}
	for _, field := range destPortFields {
		if port := lis.extractPortFromValue(data, field); port > 0 {
			parsed.DestPort = port
			break
		}
	}

	// Protocol fields
	protocolFields := []string{"protocol", "proto", "type", "service"}
	for _, field := range protocolFields {
		if val, ok := data[field].(string); ok {
			parsed.Protocol = strings.ToUpper(val)
			break
		}
	}

	// Handle nested structures
	nestedFields := []string{"source", "src", "destination", "dst", "client", "server", "network", "event"}
	for _, field := range nestedFields {
		if nested, ok := data[field].(map[string]interface{}); ok {
			lis.extractFromNested(nested, &parsed)
		}
	}

	// Store remaining fields as extra
	for k, v := range data {
		parsed.Extra[k] = v
	}

	return parsed
}

// extractIPFromValue extracts IP address from a JSON field value
func (lis *LogIngestionScanner) extractIPFromValue(data map[string]interface{}, field string) net.IP {
	if val, ok := data[field]; ok {
		switch v := val.(type) {
		case string:
			return net.ParseIP(v)
		case map[string]interface{}:
			// Handle nested objects like {"address": "192.168.1.1"}
			if addr, ok := v["address"].(string); ok {
				return net.ParseIP(addr)
			}
			if addr, ok := v["ip"].(string); ok {
				return net.ParseIP(addr)
			}
		}
	}
	return nil
}

// extractPortFromValue extracts port number from a JSON field value
func (lis *LogIngestionScanner) extractPortFromValue(data map[string]interface{}, field string) int {
	if val, ok := data[field]; ok {
		switch v := val.(type) {
		case float64:
			port := int(v)
			if port > 0 && port <= 65535 {
				return port
			}
		case int:
			if v > 0 && v <= 65535 {
				return v
			}
		case string:
			if port, err := strconv.Atoi(v); err == nil && port > 0 && port <= 65535 {
				return port
			}
		}
	}
	return 0
}

// extractFromNested extracts fields from nested JSON structures
func (lis *LogIngestionScanner) extractFromNested(nested map[string]interface{}, parsed *JSONLogParsedData) {
	// Try to extract IP from nested
	if parsed.SourceIP == nil {
		if ip := lis.extractIPFromValue(nested, "ip"); ip != nil {
			parsed.SourceIP = ip
		} else if ip := lis.extractIPFromValue(nested, "address"); ip != nil {
			parsed.SourceIP = ip
		}
	}

	// Try to extract port from nested
	if parsed.SourcePort == 0 {
		if port := lis.extractPortFromValue(nested, "port"); port > 0 {
			parsed.SourcePort = port
		}
	}

	// Try to extract protocol from nested
	if parsed.Protocol == "" {
		if proto, ok := nested["protocol"].(string); ok {
			parsed.Protocol = strings.ToUpper(proto)
		}
	}
}

// parseTimestampValue converts various timestamp formats to time.Time
func (lis *LogIngestionScanner) parseTimestampValue(val interface{}) time.Time {
	switch v := val.(type) {
	case string:
		return lis.parseTimestamp(v)
	case float64:
		// Unix timestamp (seconds or milliseconds)
		if v > 1e12 {
			// Milliseconds
			return time.Unix(int64(v/1000), int64(v)%1000*1e6)
		}
		return time.Unix(int64(v), 0)
	case int64:
		if v > 1e12 {
			return time.Unix(v/1000, v%1000*1e6)
		}
		return time.Unix(v, 0)
	}
	return time.Time{}
}

// parseGenericLog parses generic log format using regex extraction.
func (lis *LogIngestionScanner) parseGenericLog(line string) *PacketInfo {
	if line == "" {
		return nil
	}

	var sourceIP, destIP net.IP
	var sourcePort, destPort int
	timestamp := time.Now()
	protocol := "UNKNOWN"

	// Extract all IP addresses from the line
	ipMatches := ipv4Pattern.FindAllStringSubmatch(line, -1)
	if len(ipMatches) > 0 {
		sourceIP = net.ParseIP(ipMatches[0][1])
		if len(ipMatches) > 1 {
			destIP = net.ParseIP(ipMatches[1][1])
		}
	}

	// Try IPv6 if no IPv4 found
	if sourceIP == nil {
		ipv6Matches := ipv6Pattern.FindAllStringSubmatch(line, -1)
		if len(ipv6Matches) > 0 {
			sourceIP = net.ParseIP(ipv6Matches[0][1])
			if len(ipv6Matches) > 1 {
				destIP = net.ParseIP(ipv6Matches[1][1])
			}
		}
	}

	// Extract port numbers
	portMatches := portPattern.FindAllStringSubmatch(line, -1)
	for i, match := range portMatches {
		for _, p := range match[1:] {
			if p != "" {
				if port, err := strconv.Atoi(p); err == nil && port > 0 && port <= 65535 {
					if i == 0 && sourcePort == 0 {
						sourcePort = port
					} else if destPort == 0 {
						destPort = port
					}
					break
				}
			}
		}
	}

	// Try to extract timestamp using various patterns
	for _, pattern := range timestampPatterns {
		if matches := pattern.FindStringSubmatch(line); len(matches) > 1 {
			if ts := lis.parseTimestamp(matches[1]); !ts.IsZero() {
				timestamp = ts
				break
			}
		}
	}

	// Try to determine protocol from log content
	lineLower := strings.ToLower(line)
	protocolKeywords := map[string]string{
		"ssh":   "SSH",
		"http":  "HTTP",
		"https": "HTTPS",
		"ftp":   "FTP",
		"smtp":  "SMTP",
		"dns":   "DNS",
		"tcp":   "TCP",
		"udp":   "UDP",
		"icmp":  "ICMP",
		"sql":   "SQL",
		"mysql": "MySQL",
		"pgsql": "PostgreSQL",
		"redis": "Redis",
		"ldap":  "LDAP",
	}

	for keyword, proto := range protocolKeywords {
		if strings.Contains(lineLower, keyword) {
			protocol = proto
			break
		}
	}

	// Detect common well-known ports and set protocol if not already determined
	if protocol == "UNKNOWN" {
		switch destPort {
		case 22:
			protocol = "SSH"
		case 80:
			protocol = "HTTP"
		case 443:
			protocol = "HTTPS"
		case 21:
			protocol = "FTP"
		case 25, 587:
			protocol = "SMTP"
		case 53:
			protocol = "DNS"
		case 3306:
			protocol = "MySQL"
		case 5432:
			protocol = "PostgreSQL"
		case 6379:
			protocol = "Redis"
		case 27017:
			protocol = "MongoDB"
		}
	}

	return &PacketInfo{
		SourceIP:   sourceIP,
		DestIP:     destIP,
		SourcePort: sourcePort,
		DestPort:   destPort,
		Protocol:   protocol,
		Size:       len(line),
		Timestamp:  timestamp,
		Payload:    []byte(line),
	}
}

// parseTimestamp attempts to parse a timestamp string in various formats
func (lis *LogIngestionScanner) parseTimestamp(ts string) time.Time {
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000000Z",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.000",
		"2006-01-02 15:04:05,000",
		"02/Jan/2006:15:04:05 -0700",
		"Jan 2 15:04:05",
		"Jan  2 15:04:05",
		"Mon Jan 2 15:04:05 2006",
		"Mon Jan 2 15:04:05 MST 2006",
		"02-Jan-2006 15:04:05",
		"2006/01/02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, ts); err == nil {
			// If year is 0 (e.g., syslog format), add current year
			if t.Year() == 0 {
				t = t.AddDate(time.Now().Year(), 0, 0)
			}
			return t
		}
	}

	return time.Time{}
}
