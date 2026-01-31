// Package gagachat implements linguistic steganography for secure communication.
// It hides secret messages within natural-looking text using various encoding techniques.
package gagachat

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode"
)

// EncodingMethod represents the steganographic encoding technique
type EncodingMethod string

const (
	// MethodZeroWidth uses zero-width characters between visible characters
	MethodZeroWidth EncodingMethod = "zero_width"
	// MethodSynonym replaces words with synonyms based on bit values
	MethodSynonym EncodingMethod = "synonym"
	// MethodWhitespace uses variable whitespace patterns
	MethodWhitespace EncodingMethod = "whitespace"
	// MethodPunctuation uses punctuation variations
	MethodPunctuation EncodingMethod = "punctuation"
	// MethodHybrid combines multiple methods
	MethodHybrid EncodingMethod = "hybrid"
)

// Zero-width characters for encoding
const (
	zwsp = '\u200B' // Zero-width space (bit 0)
	zwnj = '\u200C' // Zero-width non-joiner (bit 1)
	zwj  = '\u200D' // Zero-width joiner (separator)
	wj   = '\u2060' // Word joiner (end marker)
)

// Message represents a steganographic message
type Message struct {
	ID          string
	CoverText   string
	SecretData  []byte
	EncodedText string
	Method      EncodingMethod
	Encrypted   bool
	Timestamp   int64
}

// Encoder handles steganographic encoding
type Encoder struct {
	mu            sync.RWMutex
	method        EncodingMethod
	encryptionKey []byte
	synonymMap    map[string][]string
}

// EncoderConfig configures the encoder
type EncoderConfig struct {
	Method        EncodingMethod
	EncryptionKey string
}

// DefaultEncoderConfig returns default configuration
func DefaultEncoderConfig() EncoderConfig {
	return EncoderConfig{
		Method: MethodZeroWidth,
	}
}

// NewEncoder creates a new steganographic encoder
func NewEncoder(cfg EncoderConfig) *Encoder {
	e := &Encoder{
		method:     cfg.Method,
		synonymMap: buildSynonymMap(),
	}

	if cfg.EncryptionKey != "" {
		hash := sha256.Sum256([]byte(cfg.EncryptionKey))
		e.encryptionKey = hash[:]
	}

	return e
}

// Encode hides secret data within cover text
func (e *Encoder) Encode(coverText string, secretData []byte) (*Message, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(coverText) == 0 {
		return nil, errors.New("cover text cannot be empty")
	}

	if len(secretData) == 0 {
		return nil, errors.New("secret data cannot be empty")
	}

	// Encrypt if key is set
	dataToEncode := secretData
	encrypted := false
	if e.encryptionKey != nil {
		var err error
		dataToEncode, err = encrypt(secretData, e.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("encryption failed: %w", err)
		}
		encrypted = true
	}

	var encodedText string
	var err error

	switch e.method {
	case MethodZeroWidth:
		encodedText, err = e.encodeZeroWidth(coverText, dataToEncode)
	case MethodSynonym:
		encodedText, err = e.encodeSynonym(coverText, dataToEncode)
	case MethodWhitespace:
		encodedText, err = e.encodeWhitespace(coverText, dataToEncode)
	case MethodPunctuation:
		encodedText, err = e.encodePunctuation(coverText, dataToEncode)
	case MethodHybrid:
		encodedText, err = e.encodeHybrid(coverText, dataToEncode)
	default:
		return nil, fmt.Errorf("unknown encoding method: %s", e.method)
	}

	if err != nil {
		return nil, err
	}

	return &Message{
		ID:          generateID(),
		CoverText:   coverText,
		SecretData:  secretData,
		EncodedText: encodedText,
		Method:      e.method,
		Encrypted:   encrypted,
	}, nil
}

// Decode extracts secret data from encoded text
func (e *Encoder) Decode(encodedText string) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(encodedText) == 0 {
		return nil, errors.New("encoded text cannot be empty")
	}

	var data []byte
	var err error

	switch e.method {
	case MethodZeroWidth:
		data, err = e.decodeZeroWidth(encodedText)
	case MethodSynonym:
		data, err = e.decodeSynonym(encodedText)
	case MethodWhitespace:
		data, err = e.decodeWhitespace(encodedText)
	case MethodPunctuation:
		data, err = e.decodePunctuation(encodedText)
	case MethodHybrid:
		data, err = e.decodeHybrid(encodedText)
	default:
		return nil, fmt.Errorf("unknown encoding method: %s", e.method)
	}

	if err != nil {
		return nil, err
	}

	// Decrypt if key is set
	if e.encryptionKey != nil {
		data, err = decrypt(data, e.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("decryption failed: %w", err)
		}
	}

	return data, nil
}

// SetMethod changes the encoding method
func (e *Encoder) SetMethod(method EncodingMethod) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.method = method
}

// SetEncryptionKey sets or updates the encryption key
func (e *Encoder) SetEncryptionKey(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if key == "" {
		e.encryptionKey = nil
		return
	}

	hash := sha256.Sum256([]byte(key))
	e.encryptionKey = hash[:]
}

// GetCapacity returns how much data can be hidden in the cover text
func (e *Encoder) GetCapacity(coverText string, method EncodingMethod) int {
	switch method {
	case MethodZeroWidth:
		// Can encode 1 bit per character position
		return len(coverText) / 8
	case MethodSynonym:
		// Depends on number of words with synonyms
		words := strings.Fields(coverText)
		count := 0
		for _, word := range words {
			if _, exists := e.synonymMap[strings.ToLower(word)]; exists {
				count++
			}
		}
		return count / 8
	case MethodWhitespace:
		// Can encode in spaces between words
		return strings.Count(coverText, " ") / 8
	case MethodPunctuation:
		// Limited capacity
		return 4
	case MethodHybrid:
		return e.GetCapacity(coverText, MethodZeroWidth) +
			e.GetCapacity(coverText, MethodSynonym)
	default:
		return 0
	}
}

// Zero-width encoding
func (e *Encoder) encodeZeroWidth(cover string, data []byte) (string, error) {
	// Convert data to binary string
	binary := dataToBinary(data)

	var result strings.Builder
	binaryIndex := 0

	// Insert zero-width characters after each visible character
	for _, char := range cover {
		result.WriteRune(char)

		if binaryIndex < len(binary) && unicode.IsLetter(char) {
			if binary[binaryIndex] == '0' {
				result.WriteRune(zwsp)
			} else {
				result.WriteRune(zwnj)
			}
			binaryIndex++
		}
	}

	// Add end marker
	result.WriteRune(wj)

	if binaryIndex < len(binary) {
		return "", errors.New("cover text too short for secret data")
	}

	return result.String(), nil
}

func (e *Encoder) decodeZeroWidth(encoded string) ([]byte, error) {
	var binary strings.Builder
	foundEnd := false

	for _, char := range encoded {
		switch char {
		case zwsp:
			binary.WriteByte('0')
		case zwnj:
			binary.WriteByte('1')
		case wj:
			foundEnd = true
		}

		if foundEnd {
			break
		}
	}

	if binary.Len() == 0 {
		return nil, errors.New("no hidden data found")
	}

	return binaryToData(binary.String())
}

// Synonym encoding
func (e *Encoder) encodeSynonym(cover string, data []byte) (string, error) {
	binary := dataToBinary(data)
	words := strings.Fields(cover)
	binaryIndex := 0

	for i, word := range words {
		if binaryIndex >= len(binary) {
			break
		}

		lowerWord := strings.ToLower(word)
		if synonyms, exists := e.synonymMap[lowerWord]; exists && len(synonyms) > 1 {
			// Use original word for 0, first synonym for 1
			if binary[binaryIndex] == '1' {
				// Preserve case
				if unicode.IsUpper([]rune(word)[0]) {
					words[i] = strings.Title(synonyms[0])
				} else {
					words[i] = synonyms[0]
				}
			}
			binaryIndex++
		}
	}

	if binaryIndex < len(binary) {
		return "", errors.New("cover text too short for secret data")
	}

	return strings.Join(words, " "), nil
}

func (e *Encoder) decodeSynonym(encoded string) ([]byte, error) {
	words := strings.Fields(encoded)
	var binary strings.Builder

	for _, word := range words {
		lowerWord := strings.ToLower(word)

		// Check if word is a synonym (bit 1)
		for original, synonyms := range e.synonymMap {
			for _, syn := range synonyms {
				if syn == lowerWord {
					binary.WriteByte('1')
					goto next
				}
			}
			if original == lowerWord {
				binary.WriteByte('0')
				goto next
			}
		}
	next:
	}

	if binary.Len() == 0 {
		return nil, errors.New("no hidden data found")
	}

	return binaryToData(binary.String())
}

// Whitespace encoding
func (e *Encoder) encodeWhitespace(cover string, data []byte) (string, error) {
	binary := dataToBinary(data)
	words := strings.Fields(cover)
	binaryIndex := 0

	var result strings.Builder
	for i, word := range words {
		result.WriteString(word)

		if i < len(words)-1 && binaryIndex < len(binary) {
			// One space for 0, two spaces for 1
			if binary[binaryIndex] == '0' {
				result.WriteByte(' ')
			} else {
				result.WriteString("  ")
			}
			binaryIndex++
		} else if i < len(words)-1 {
			result.WriteByte(' ')
		}
	}

	if binaryIndex < len(binary) {
		return "", errors.New("cover text too short for secret data")
	}

	return result.String(), nil
}

func (e *Encoder) decodeWhitespace(encoded string) ([]byte, error) {
	var binary strings.Builder
	inWord := false
	spaceCount := 0

	for _, char := range encoded {
		if unicode.IsSpace(char) {
			if inWord {
				inWord = false
				spaceCount = 1
			} else {
				spaceCount++
			}
		} else {
			if !inWord && spaceCount > 0 {
				if spaceCount == 1 {
					binary.WriteByte('0')
				} else {
					binary.WriteByte('1')
				}
				spaceCount = 0
			}
			inWord = true
		}
	}

	if binary.Len() == 0 {
		return nil, errors.New("no hidden data found")
	}

	return binaryToData(binary.String())
}

// Punctuation encoding
func (e *Encoder) encodePunctuation(cover string, data []byte) (string, error) {
	// Limited encoding using period/comma patterns
	binary := dataToBinary(data)

	if len(binary) > 32 { // Limit to 4 bytes
		return "", errors.New("punctuation method supports max 4 bytes")
	}

	result := cover

	// Append encoded data as subtle punctuation pattern at end
	var encoded strings.Builder
	for i := 0; i < len(binary); i += 2 {
		bits := binary[i:min(i+2, len(binary))]
		switch bits {
		case "00":
			encoded.WriteString(".. ")
		case "01":
			encoded.WriteString("... ")
		case "10":
			encoded.WriteString(".... ")
		case "11":
			encoded.WriteString("..... ")
		}
	}

	return result + " " + strings.TrimSpace(encoded.String()), nil
}

func (e *Encoder) decodePunctuation(encoded string) ([]byte, error) {
	var binary strings.Builder

	// Find punctuation pattern at end
	parts := strings.Split(encoded, " ")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, ".") && !strings.Contains(part, "a") {
			dotCount := strings.Count(part, ".")
			switch dotCount {
			case 2:
				binary.WriteString("00")
			case 3:
				binary.WriteString("01")
			case 4:
				binary.WriteString("10")
			case 5:
				binary.WriteString("11")
			}
		}
	}

	if binary.Len() == 0 {
		return nil, errors.New("no hidden data found")
	}

	return binaryToData(binary.String())
}

// Hybrid encoding
func (e *Encoder) encodeHybrid(cover string, data []byte) (string, error) {
	// Use zero-width for first half, synonym for second half
	mid := len(data) / 2
	if mid == 0 {
		mid = 1
	}

	// First pass: zero-width
	intermediate, err := e.encodeZeroWidth(cover, data[:mid])
	if err != nil {
		return "", err
	}

	// Second pass: try synonym if there's remaining data
	if mid < len(data) {
		result, err := e.encodeSynonym(intermediate, data[mid:])
		if err != nil {
			// Fall back to zero-width only
			return e.encodeZeroWidth(cover, data)
		}
		return result, nil
	}

	return intermediate, nil
}

func (e *Encoder) decodeHybrid(encoded string) ([]byte, error) {
	// Try to decode both methods and combine
	zwData, err1 := e.decodeZeroWidth(encoded)
	synData, err2 := e.decodeSynonym(encoded)

	if err1 != nil && err2 != nil {
		return nil, errors.New("hybrid decode failed")
	}

	if err1 == nil && err2 == nil {
		// Combine results
		result := make([]byte, 0, len(zwData)+len(synData))
		result = append(result, zwData...)
		result = append(result, synData...)
		return result, nil
	}

	if err1 == nil {
		return zwData, nil
	}
	return synData, nil
}

// Helper functions
func dataToBinary(data []byte) string {
	var binary strings.Builder
	for _, b := range data {
		binary.WriteString(fmt.Sprintf("%08b", b))
	}
	return binary.String()
}

func binaryToData(binary string) ([]byte, error) {
	// Pad to multiple of 8
	for len(binary)%8 != 0 {
		binary += "0"
	}

	data := make([]byte, len(binary)/8)
	for i := 0; i < len(binary); i += 8 {
		var b byte
		for j := 0; j < 8; j++ {
			if binary[i+j] == '1' {
				b |= 1 << (7 - j)
			}
		}
		data[i/8] = b
	}
	return data, nil
}

func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func buildSynonymMap() map[string][]string {
	return map[string][]string{
		"big":       {"large", "huge", "vast"},
		"small":     {"tiny", "little", "minor"},
		"fast":      {"quick", "rapid", "swift"},
		"slow":      {"sluggish", "gradual", "leisurely"},
		"good":      {"great", "fine", "excellent"},
		"bad":       {"poor", "terrible", "awful"},
		"happy":     {"joyful", "pleased", "content"},
		"sad":       {"unhappy", "sorrowful", "melancholy"},
		"start":     {"begin", "commence", "initiate"},
		"end":       {"finish", "conclude", "terminate"},
		"help":      {"assist", "aid", "support"},
		"use":       {"utilize", "employ", "apply"},
		"make":      {"create", "produce", "construct"},
		"get":       {"obtain", "acquire", "receive"},
		"see":       {"observe", "view", "notice"},
		"know":      {"understand", "comprehend", "realize"},
		"think":     {"believe", "consider", "suppose"},
		"come":      {"arrive", "approach", "appear"},
		"go":        {"proceed", "advance", "depart"},
		"want":      {"desire", "wish", "crave"},
		"important": {"significant", "crucial", "vital"},
		"problem":   {"issue", "difficulty", "challenge"},
		"answer":    {"response", "reply", "solution"},
		"question":  {"query", "inquiry", "matter"},
		"work":      {"labor", "effort", "task"},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Chat provides a higher-level interface for secure messaging
type Chat struct {
	encoder  *Encoder
	messages []Message
	mu       sync.RWMutex
}

// NewChat creates a new secure chat
func NewChat(encryptionKey string) *Chat {
	cfg := DefaultEncoderConfig()
	cfg.EncryptionKey = encryptionKey
	return &Chat{
		encoder:  NewEncoder(cfg),
		messages: make([]Message, 0),
	}
}

// SendSecure encodes and stores a secure message
func (c *Chat) SendSecure(coverText string, secretMessage string) (*Message, error) {
	msg, err := c.encoder.Encode(coverText, []byte(secretMessage))
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.messages = append(c.messages, *msg)
	c.mu.Unlock()

	return msg, nil
}

// ReceiveSecure decodes a message
func (c *Chat) ReceiveSecure(encodedText string) (string, error) {
	data, err := c.encoder.Decode(encodedText)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetVisibleText extracts visible text from encoded message
func (c *Chat) GetVisibleText(encodedText string) string {
	var result strings.Builder
	for _, char := range encodedText {
		// Skip zero-width characters
		if char != zwsp && char != zwnj && char != zwj && char != wj {
			result.WriteRune(char)
		}
	}
	return result.String()
}

// AnalyzeText checks if text might contain hidden data
func (c *Chat) AnalyzeText(text string) map[string]interface{} {
	zwCount := 0
	for _, char := range text {
		if char == zwsp || char == zwnj || char == zwj || char == wj {
			zwCount++
		}
	}

	doubleSpaces := strings.Count(text, "  ")
	dotPatterns := strings.Count(text, "...")

	return map[string]interface{}{
		"has_zero_width":     zwCount > 0,
		"zero_width_count":   zwCount,
		"has_double_spaces":  doubleSpaces > 0,
		"double_space_count": doubleSpaces,
		"has_dot_patterns":   dotPatterns > 0,
		"dot_pattern_count":  dotPatterns,
		"likely_stego":       zwCount > 5 || doubleSpaces > 3,
		"text_length":        len(text),
		"visible_length":     len(c.GetVisibleText(text)),
	}
}

// EncodeBase64URL encodes message for safe transmission
func EncodeBase64URL(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeBase64URL decodes message from transmission format
func DecodeBase64URL(encoded string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(encoded)
}
