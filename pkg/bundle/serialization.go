package bundle

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// Encoder handles bundle serialization.
type Encoder struct {
	w io.Writer
}

// Decoder handles bundle deserialization.
type Decoder struct {
	r io.Reader
}

// NewEncoder creates a new bundle encoder.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// NewDecoder creates a new bundle decoder.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Encode serializes a bundle to the writer in binary format.
func (e *Encoder) Encode(b *Bundle) error {
	if err := b.Validate(); err != nil {
		return fmt.Errorf("invalid bundle: %w", err)
	}

	// Write version
	if err := binary.Write(e.w, binary.BigEndian, b.Version); err != nil {
		return fmt.Errorf("failed to write version: %w", err)
	}

	// Write bundle ID
	idBytes, _ := b.ID.MarshalBinary()
	if _, err := e.w.Write(idBytes); err != nil {
		return fmt.Errorf("failed to write ID: %w", err)
	}

	// Write flags
	if err := binary.Write(e.w, binary.BigEndian, b.BundleFlags); err != nil {
		return fmt.Errorf("failed to write flags: %w", err)
	}

	// Write EIDs
	if err := e.writeString(b.SourceEID); err != nil {
		return err
	}
	if err := e.writeString(b.DestinationEID); err != nil {
		return err
	}
	if err := e.writeString(b.ReportTo); err != nil {
		return err
	}

	// Write timestamps
	if err := binary.Write(e.w, binary.BigEndian, b.CreationTimestamp.UnixNano()); err != nil {
		return fmt.Errorf("failed to write timestamp: %w", err)
	}
	if err := binary.Write(e.w, binary.BigEndian, int64(b.Lifetime)); err != nil {
		return fmt.Errorf("failed to write lifetime: %w", err)
	}

	// Write metadata
	if err := binary.Write(e.w, binary.BigEndian, b.CRCType); err != nil {
		return err
	}
	if err := binary.Write(e.w, binary.BigEndian, b.HopCount); err != nil {
		return err
	}
	if err := binary.Write(e.w, binary.BigEndian, b.Priority); err != nil {
		return err
	}
	if err := e.writeString(b.PreviousNode); err != nil {
		return err
	}

	// Write fragment info
	var isFragByte uint8
	if b.IsFragment {
		isFragByte = 1
	}
	if err := binary.Write(e.w, binary.BigEndian, isFragByte); err != nil {
		return err
	}
	if err := binary.Write(e.w, binary.BigEndian, b.FragmentOffset); err != nil {
		return err
	}
	if err := binary.Write(e.w, binary.BigEndian, b.TotalADULength); err != nil {
		return err
	}

	// Write payload
	if err := binary.Write(e.w, binary.BigEndian, uint32(len(b.Payload))); err != nil {
		return fmt.Errorf("failed to write payload length: %w", err)
	}
	if _, err := e.w.Write(b.Payload); err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}

	return nil
}

func (e *Encoder) writeString(s string) error {
	data := []byte(s)
	if err := binary.Write(e.w, binary.BigEndian, uint16(len(data))); err != nil {
		return fmt.Errorf("failed to write string length: %w", err)
	}
	if _, err := e.w.Write(data); err != nil {
		return fmt.Errorf("failed to write string: %w", err)
	}
	return nil
}

// Decode deserializes a bundle from the reader.
func (d *Decoder) Decode() (*Bundle, error) {
	b := &Bundle{}

	// Read version
	if err := binary.Read(d.r, binary.BigEndian, &b.Version); err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}

	// Read bundle ID
	idBytes := make([]byte, 16)
	if _, err := io.ReadFull(d.r, idBytes); err != nil {
		return nil, fmt.Errorf("failed to read ID: %w", err)
	}
	b.ID, _ = uuid.FromBytes(idBytes)

	// Read flags
	if err := binary.Read(d.r, binary.BigEndian, &b.BundleFlags); err != nil {
		return nil, fmt.Errorf("failed to read flags: %w", err)
	}

	// Read EIDs
	var err error
	b.SourceEID, err = d.readString()
	if err != nil {
		return nil, err
	}
	b.DestinationEID, err = d.readString()
	if err != nil {
		return nil, err
	}
	b.ReportTo, err = d.readString()
	if err != nil {
		return nil, err
	}

	// Read timestamps
	var tsNano int64
	if err := binary.Read(d.r, binary.BigEndian, &tsNano); err != nil {
		return nil, fmt.Errorf("failed to read timestamp: %w", err)
	}
	b.CreationTimestamp = time.Unix(0, tsNano)

	var lifetimeNano int64
	if err := binary.Read(d.r, binary.BigEndian, &lifetimeNano); err != nil {
		return nil, fmt.Errorf("failed to read lifetime: %w", err)
	}
	b.Lifetime = time.Duration(lifetimeNano)

	// Read metadata
	if err := binary.Read(d.r, binary.BigEndian, &b.CRCType); err != nil {
		return nil, err
	}
	if err := binary.Read(d.r, binary.BigEndian, &b.HopCount); err != nil {
		return nil, err
	}
	if err := binary.Read(d.r, binary.BigEndian, &b.Priority); err != nil {
		return nil, err
	}
	b.PreviousNode, err = d.readString()
	if err != nil {
		return nil, err
	}

	// Read fragment info
	var isFragByte uint8
	if err := binary.Read(d.r, binary.BigEndian, &isFragByte); err != nil {
		return nil, err
	}
	b.IsFragment = isFragByte == 1
	if err := binary.Read(d.r, binary.BigEndian, &b.FragmentOffset); err != nil {
		return nil, err
	}
	if err := binary.Read(d.r, binary.BigEndian, &b.TotalADULength); err != nil {
		return nil, err
	}

	// Read payload
	var payloadLen uint32
	if err := binary.Read(d.r, binary.BigEndian, &payloadLen); err != nil {
		return nil, fmt.Errorf("failed to read payload length: %w", err)
	}
	b.Payload = make([]byte, payloadLen)
	if _, err := io.ReadFull(d.r, b.Payload); err != nil {
		return nil, fmt.Errorf("failed to read payload: %w", err)
	}

	return b, nil
}

func (d *Decoder) readString() (string, error) {
	var length uint16
	if err := binary.Read(d.r, binary.BigEndian, &length); err != nil {
		return "", fmt.Errorf("failed to read string length: %w", err)
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(d.r, data); err != nil {
		return "", fmt.Errorf("failed to read string: %w", err)
	}
	return string(data), nil
}

// Marshal serializes a bundle to bytes.
func Marshal(b *Bundle) ([]byte, error) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(b); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal deserializes a bundle from bytes.
func Unmarshal(data []byte) (*Bundle, error) {
	dec := NewDecoder(bytes.NewReader(data))
	return dec.Decode()
}

// MarshalJSON serializes a bundle to JSON format.
func (b *Bundle) MarshalJSON() ([]byte, error) {
	type bundleJSON struct {
		ID                string `json:"id"`
		Version           uint8  `json:"version"`
		BundleFlags       uint64 `json:"bundleFlags"`
		DestinationEID    string `json:"destinationEid"`
		SourceEID         string `json:"sourceEid"`
		ReportTo          string `json:"reportTo"`
		CreationTimestamp string `json:"creationTimestamp"`
		Lifetime          string `json:"lifetime"`
		Payload           []byte `json:"payload"`
		CRCType           uint8  `json:"crcType"`
		PreviousNode      string `json:"previousNode"`
		HopCount          uint32 `json:"hopCount"`
		Priority          uint8  `json:"priority"`
		FragmentOffset    uint64 `json:"fragmentOffset,omitempty"`
		TotalADULength    uint64 `json:"totalAduLength,omitempty"`
		IsFragment        bool   `json:"isFragment"`
	}

	return json.Marshal(bundleJSON{
		ID:                b.ID.String(),
		Version:           b.Version,
		BundleFlags:       b.BundleFlags,
		DestinationEID:    b.DestinationEID,
		SourceEID:         b.SourceEID,
		ReportTo:          b.ReportTo,
		CreationTimestamp: b.CreationTimestamp.Format(time.RFC3339Nano),
		Lifetime:          b.Lifetime.String(),
		Payload:           b.Payload,
		CRCType:           b.CRCType,
		PreviousNode:      b.PreviousNode,
		HopCount:          b.HopCount,
		Priority:          b.Priority,
		FragmentOffset:    b.FragmentOffset,
		TotalADULength:    b.TotalADULength,
		IsFragment:        b.IsFragment,
	})
}
