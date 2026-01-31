package actuators

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sync"
	"time"

	"go.bug.st/serial"
	"go.bug.st/serial/enumerator"
)

// MAVLinkProtocol implements the MAVLink v2.0 protocol
type MAVLinkProtocol struct {
	port     serial.Port
	mu       sync.RWMutex
	sequence uint8
	systemID uint8
	compID   uint8
}

// MAVLinkMessage represents a MAVLink message
type MAVLinkMessage struct {
	Magic       uint8
	Length      uint8
	Incompat    uint8
	Compat      uint8
	Sequence    uint8
	SystemID    uint8
	ComponentID uint8
	MessageID   uint32
	Payload     []byte
	Checksum    uint16
}

// MAVLink message IDs
const (
	MAVLINK_MSG_ID_HEARTBEAT                      = 0
	MAVLINK_MSG_ID_SYS_STATUS                     = 1
	MAVLINK_MSG_ID_SYSTEM_TIME                    = 2
	MAVLINK_MSG_ID_PING                           = 4
	MAVLINK_MSG_ID_CHANGE_OPERATOR_CONTROL        = 5
	MAVLINK_MSG_ID_CHANGE_OPERATOR_CONTROL_ACK    = 6
	MAVLINK_MSG_ID_ATTITUDE                       = 30
	MAVLINK_MSG_ID_ATTITUDE_QUATERNION            = 31
	MAVLINK_MSG_ID_LOCAL_POSITION_NED             = 32
	MAVLINK_MSG_ID_GLOBAL_POSITION_INT            = 33
	MAVLINK_MSG_ID_RC_CHANNELS                    = 65
	MAVLINK_MSG_ID_REQUEST_DATA_STREAM            = 66
	MAVLINK_MSG_ID_MISSION_REQUEST_LIST           = 43
	MAVLINK_MSG_ID_MISSION_COUNT                  = 44
	MAVLINK_MSG_ID_MISSION_REQUEST                = 40
	MAVLINK_MSG_ID_MISSION_ACK                    = 47
	MAVLINK_MSG_ID_SET_MODE                       = 11
	MAVLINK_MSG_ID_PARAM_REQUEST_LIST             = 21
	MAVLINK_MSG_ID_PARAM_VALUE                    = 22
	MAVLINK_MSG_ID_SET_PARAM                      = 23
	MAVLINK_MSG_ID_GPS_RAW_INT                    = 24
	MAVLINK_MSG_ID_GPS_STATUS                     = 25
	MAVLINK_MSG_ID_SCALED_IMU                     = 26
	MAVLINK_MSG_ID_RAW_IMU                        = 27
	MAVLINK_MSG_ID_RAW_PRESSURE                   = 28
	MAVLINK_MSG_ID_SCALED_PRESSURE                = 29
	MAVLINK_MSG_ID_SET_ATTITUDE_TARGET            = 82
	MAVLINK_MSG_ID_SET_POSITION_TARGET_LOCAL_NED  = 84
	MAVLINK_MSG_ID_SET_POSITION_TARGET_GLOBAL_INT = 86
	MAVLINK_MSG_ID_COMMAND_LONG                   = 76
	MAVLINK_MSG_ID_COMMAND_ACK                    = 77
)

// MAVLink commands
const (
	MAV_CMD_COMPONENT_ARM_DISARM = 400
	MAV_CMD_NAV_RETURN_TO_LAUNCH = 20
	MAV_CMD_NAV_LAND             = 21
	MAV_CMD_DO_SET_MODE          = 176
)

// MAVLink modes
const (
	MAV_MODE_PREFLIGHT          = 0
	MAV_MODE_MANUAL_DISARMED    = 64
	MAV_MODE_TEST_DISARMED      = 66
	MAV_MODE_STABILIZE_DISARMED = 80
	MAV_MODE_GUIDED_DISARMED    = 88
	MAV_MODE_AUTO_DISARMED      = 92
	MAV_MODE_MANUAL_ARMED       = 192
	MAV_MODE_TEST_ARMED         = 194
	MAV_MODE_STABILIZE_ARMED    = 208
	MAV_MODE_GUIDED_ARMED       = 216
	MAV_MODE_AUTO_ARMED         = 220
)

// MAVLink frame types
const (
	MAVLINK_FRAME_GLOBAL_RELATIVE_ALT = 3
	MAVLINK_FRAME_LOCAL_NED           = 1
)

// MAVLink magic byte
const MAVLINK_V2_MAGIC = 0xFD

// NewMAVLinkProtocol creates a new MAVLink protocol handler
func NewMAVLinkProtocol(systemID, compID uint8) *MAVLinkProtocol {
	return &MAVLinkProtocol{
		sequence: 0,
		systemID: systemID,
		compID:   compID,
	}
}

// OpenSerialPort opens a serial port for MAVLink communication
func (mp *MAVLinkProtocol) OpenSerialPort(portName string, baudRate int) error {
	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return fmt.Errorf("failed to open serial port %s: %w", portName, err)
	}

	mp.mu.Lock()
	mp.port = port
	mp.mu.Unlock()

	return nil
}

// Close closes the serial port
func (mp *MAVLinkProtocol) Close() error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.port != nil {
		return mp.port.Close()
	}
	return nil
}

// SendHeartbeat sends a MAVLink heartbeat message
func (mp *MAVLinkProtocol) SendHeartbeat(autopilot, baseMode, customMode, systemStatus uint8) error {
	payload := make([]byte, 9)
	payload[0] = autopilot
	payload[1] = baseMode
	binary.LittleEndian.PutUint32(payload[2:6], uint32(customMode))
	payload[6] = systemStatus
	payload[7] = 3 // MAVLink version
	payload[8] = 0 // Vehicle type (generic)

	return mp.sendMessage(MAVLINK_MSG_ID_HEARTBEAT, payload)
}

// SendCommandLong sends a MAVLink command_long message
func (mp *MAVLinkProtocol) SendCommandLong(targetSystem, targetComponent uint8, command uint16, confirmation uint8, params [7]float32) error {
	payload := make([]byte, 33)
	payload[0] = targetSystem
	payload[1] = targetComponent
	binary.LittleEndian.PutUint16(payload[2:4], command)
	payload[4] = confirmation

	offset := 5
	for i := 0; i < 7; i++ {
		binary.LittleEndian.PutUint32(payload[offset:offset+4], math.Float32bits(params[i]))
		offset += 4
	}

	return mp.sendMessage(MAVLINK_MSG_ID_COMMAND_LONG, payload)
}

// SendSetMode sends a MAVLink set_mode message
func (mp *MAVLinkProtocol) SendSetMode(targetSystem, baseMode, customMode uint8) error {
	payload := make([]byte, 4)
	payload[0] = targetSystem
	payload[1] = baseMode
	binary.LittleEndian.PutUint32(payload[2:6], uint32(customMode))

	return mp.sendMessage(MAVLINK_MSG_ID_SET_MODE, payload)
}

// SendSetAttitudeTarget sends attitude setpoint
func (mp *MAVLinkProtocol) SendSetAttitudeTarget(targetSystem, targetComponent uint8, timeBootMs uint32, typeMask uint8, q [4]float32, bodyRollRate, bodyPitchRate, bodyYawRate float32, thrust float32) error {
	payload := make([]byte, 37)
	payload[0] = byte(timeBootMs & 0xFF)
	payload[1] = byte((timeBootMs >> 8) & 0xFF)
	payload[2] = byte((timeBootMs >> 16) & 0xFF)
	payload[3] = byte((timeBootMs >> 24) & 0xFF)
	payload[4] = targetSystem
	payload[5] = targetComponent
	payload[6] = typeMask

	offset := 7
	for i := 0; i < 4; i++ {
		binary.LittleEndian.PutUint32(payload[offset:offset+4], math.Float32bits(q[i]))
		offset += 4
	}

	binary.LittleEndian.PutUint32(payload[offset:offset+4], math.Float32bits(bodyRollRate))
	offset += 4
	binary.LittleEndian.PutUint32(payload[offset:offset+4], math.Float32bits(bodyPitchRate))
	offset += 4
	binary.LittleEndian.PutUint32(payload[offset:offset+4], math.Float32bits(bodyYawRate))
	offset += 4
	binary.LittleEndian.PutUint32(payload[offset:offset+4], math.Float32bits(thrust))

	return mp.sendMessage(MAVLINK_MSG_ID_SET_ATTITUDE_TARGET, payload)
}

// SendSetPositionTargetLocalNED sends position/velocity setpoint
func (mp *MAVLinkProtocol) SendSetPositionTargetLocalNED(targetSystem, targetComponent uint8, timeBootMs uint32, coordinateFrame uint8, typeMask uint16, x, y, z, vx, vy, vz, afx, afy, afz, yaw, yawRate float32) error {
	payload := make([]byte, 51)
	payload[0] = byte(timeBootMs & 0xFF)
	payload[1] = byte((timeBootMs >> 8) & 0xFF)
	payload[2] = byte((timeBootMs >> 16) & 0xFF)
	payload[3] = byte((timeBootMs >> 24) & 0xFF)
	payload[4] = targetSystem
	payload[5] = targetComponent
	payload[6] = coordinateFrame
	binary.LittleEndian.PutUint16(payload[7:9], typeMask)

	offset := 9
	values := []float32{x, y, z, vx, vy, vz, afx, afy, afz, yaw, yawRate}
	for _, val := range values {
		binary.LittleEndian.PutUint32(payload[offset:offset+4], math.Float32bits(val))
		offset += 4
	}

	return mp.sendMessage(MAVLINK_MSG_ID_SET_POSITION_TARGET_LOCAL_NED, payload)
}

// sendMessage sends a MAVLink v2 message
func (mp *MAVLinkProtocol) sendMessage(messageID uint32, payload []byte) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.port == nil {
		return fmt.Errorf("serial port not open")
	}

	msg := &MAVLinkMessage{
		Magic:       MAVLINK_V2_MAGIC,
		Length:      uint8(len(payload)),
		Incompat:    0,
		Compat:      0,
		Sequence:    mp.sequence,
		SystemID:    mp.systemID,
		ComponentID: mp.compID,
		MessageID:   messageID,
		Payload:     payload,
	}

	mp.sequence++

	// Serialize message
	buf := mp.serializeMessage(msg)

	// Write to serial port
	_, err := mp.port.Write(buf)
	return err
}

// serializeMessage serializes a MAVLink message to bytes
func (mp *MAVLinkProtocol) serializeMessage(msg *MAVLinkMessage) []byte {
	buf := new(bytes.Buffer)

	buf.WriteByte(msg.Magic)
	buf.WriteByte(msg.Length)
	buf.WriteByte(msg.Incompat)
	buf.WriteByte(msg.Compat)
	buf.WriteByte(msg.Sequence)
	buf.WriteByte(msg.SystemID)
	buf.WriteByte(msg.ComponentID)

	// Message ID (3 bytes, little endian)
	idBytes := make([]byte, 3)
	binary.LittleEndian.PutUint32(idBytes, msg.MessageID)
	buf.Write(idBytes[:3])

	// Payload
	buf.Write(msg.Payload)

	// Checksum
	checksum := mp.calculateChecksum(msg)
	buf.WriteByte(uint8(checksum & 0xFF))
	buf.WriteByte(uint8((checksum >> 8) & 0xFF))

	return buf.Bytes()
}

// calculateChecksum calculates MAVLink v2 checksum
func (mp *MAVLinkProtocol) calculateChecksum(msg *MAVLinkMessage) uint16 {
	crc := mp.crcAccumulate(0xFFFF, []byte{msg.Length, msg.Incompat, msg.Compat, msg.Sequence, msg.SystemID, msg.ComponentID})

	idBytes := make([]byte, 3)
	binary.LittleEndian.PutUint32(idBytes, msg.MessageID)
	crc = mp.crcAccumulate(crc, idBytes[:3])
	crc = mp.crcAccumulate(crc, msg.Payload)

	// MAVLink v2 CRC extra
	crcExtra := mp.getCrcExtra(msg.MessageID)
	crc = mp.crcAccumulate(crc, []byte{crcExtra})

	return crc
}

// crcAccumulate accumulates CRC
func (mp *MAVLinkProtocol) crcAccumulate(crc uint16, data []byte) uint16 {
	for _, b := range data {
		tmp := uint8(crc) ^ b
		crc = (crc >> 8) ^ crcTable[tmp]
	}
	return crc
}

// getCrcExtra returns CRC extra byte for message ID
func (mp *MAVLinkProtocol) getCrcExtra(messageID uint32) uint8 {
	// Simplified - in production, use full MAVLink CRC extra table
	crcExtras := map[uint32]uint8{
		MAVLINK_MSG_ID_HEARTBEAT:                     50,
		MAVLINK_MSG_ID_SET_ATTITUDE_TARGET:           49,
		MAVLINK_MSG_ID_SET_POSITION_TARGET_LOCAL_NED: 143,
		MAVLINK_MSG_ID_COMMAND_LONG:                  152,
		MAVLINK_MSG_ID_SET_MODE:                      89,
	}
	if extra, ok := crcExtras[messageID]; ok {
		return extra
	}
	return 0
}

// CRC table for MAVLink (X.25 CRC)
var crcTable = [256]uint16{
	0x0000, 0x1021, 0x2042, 0x3063, 0x4084, 0x50a5, 0x60c6, 0x70e7,
	0x8108, 0x9129, 0xa14a, 0xb16b, 0xc18c, 0xd1ad, 0xe1ce, 0xf1ef,
	0x1231, 0x0210, 0x3273, 0x2252, 0x52b5, 0x4294, 0x72f7, 0x62d6,
	0x9339, 0x8318, 0xb37b, 0xa35a, 0xd3bd, 0xc39c, 0xf3ff, 0xe3de,
	0x2462, 0x3443, 0x0420, 0x1401, 0x64e6, 0x74c7, 0x44a4, 0x5485,
	0xa56a, 0xb54b, 0x8528, 0x9509, 0xe5ee, 0xf5cf, 0xc5ac, 0xd58d,
	0x3653, 0x2672, 0x1611, 0x0630, 0x76d7, 0x66f6, 0x5695, 0x46b4,
	0xb75b, 0xa77a, 0x9719, 0x8738, 0xf7df, 0xe7fe, 0xd79d, 0xc7bc,
	0x48c4, 0x58e5, 0x6886, 0x78a7, 0x0840, 0x1861, 0x2802, 0x3823,
	0xc9cc, 0xd9ed, 0xe98e, 0xf9af, 0x8948, 0x9969, 0xa90a, 0xb92b,
	0x5af5, 0x4ad4, 0x7ab7, 0x6a96, 0x1a71, 0x0a50, 0x3a33, 0x2a12,
	0xdbfd, 0xcbdc, 0xfbbf, 0xeb9e, 0x9b79, 0x8b58, 0xbb3b, 0xab1a,
	0x6ca6, 0x7c87, 0x4ce4, 0x5cc5, 0x2c22, 0x3c03, 0x0c60, 0x1c41,
	0xedae, 0xfd8f, 0xcdec, 0xddcd, 0xad2a, 0xbd0b, 0x8d68, 0x9d49,
	0x7e97, 0x6eb6, 0x5ed5, 0x4ef4, 0x3e13, 0x2e32, 0x1e51, 0x0e70,
	0xff9f, 0xefbe, 0xdfdd, 0xcffc, 0xbf1b, 0xaf3a, 0x9f59, 0x8f78,
	0x9188, 0x81a9, 0xb1ca, 0xa1eb, 0xd10c, 0xc12d, 0xf14e, 0xe16f,
	0x1080, 0x00a1, 0x30c2, 0x20e3, 0x5004, 0x4025, 0x7046, 0x6067,
	0x83b9, 0x9398, 0xa3fb, 0xb3da, 0xc33d, 0xd31c, 0xe37f, 0xf35e,
	0x02b1, 0x1290, 0x22f3, 0x32d2, 0x4235, 0x5214, 0x6277, 0x7256,
	0xb5ea, 0xa5cb, 0x95a8, 0x8589, 0xf56e, 0xe54f, 0xd52c, 0xc50d,
	0x34e2, 0x24c3, 0x14a0, 0x0481, 0x7466, 0x6447, 0x5424, 0x4405,
	0xa7db, 0xb7fa, 0x8799, 0x97b8, 0xe75f, 0xf77e, 0xc71d, 0xd73c,
	0x26d3, 0x36f2, 0x0691, 0x16b0, 0x6657, 0x7676, 0x4615, 0x5634,
	0xd94c, 0xc96d, 0xf90e, 0xe92f, 0x99c8, 0x89e9, 0xb98a, 0xa9ab,
	0x5844, 0x4865, 0x7806, 0x6827, 0x18c0, 0x08e1, 0x3882, 0x28a3,
	0xcb7d, 0xdb5c, 0xeb3f, 0xfb1e, 0x8bf9, 0x9bd8, 0xabbb, 0xbb9a,
	0x4a75, 0x5a54, 0x6a37, 0x7a16, 0x0af1, 0x1ad0, 0x2ab3, 0x3a92,
	0xfd2e, 0xed0f, 0xdd6c, 0xcd4d, 0xbdaa, 0xad8b, 0x9de8, 0x8dc9,
	0x7c26, 0x6c07, 0x5c64, 0x4c45, 0x3ca2, 0x2c83, 0x1ce0, 0x0cc1,
	0xef1f, 0xff3e, 0xcf5d, 0xdf7c, 0xaf9b, 0xbfba, 0x8fd9, 0x9ff8,
	0x6e17, 0x7e36, 0x4e55, 0x5e74, 0x2e93, 0x3eb2, 0x0ed1, 0x1ef0,
}

// ReadMessage reads a MAVLink message from the serial port
func (mp *MAVLinkProtocol) ReadMessage(timeout time.Duration) (*MAVLinkMessage, error) {
	mp.mu.RLock()
	port := mp.port
	mp.mu.RUnlock()

	if port == nil {
		return nil, fmt.Errorf("serial port not open")
	}

	// Set read timeout
	port.SetReadTimeout(timeout)

	// Read magic byte
	magic := make([]byte, 1)
	if _, err := port.Read(magic); err != nil {
		return nil, err
	}

	if magic[0] != MAVLINK_V2_MAGIC {
		return nil, fmt.Errorf("invalid magic byte: 0x%02x", magic[0])
	}

	// Read header
	header := make([]byte, 9)
	if _, err := io.ReadFull(port, header); err != nil {
		return nil, err
	}

	msg := &MAVLinkMessage{
		Magic:       magic[0],
		Length:      header[0],
		Incompat:    header[1],
		Compat:      header[2],
		Sequence:    header[3],
		SystemID:    header[4],
		ComponentID: header[5],
	}

	// Read message ID (3 bytes)
	msg.MessageID = uint32(header[6]) | uint32(header[7])<<8 | uint32(header[8])<<16

	// Read payload
	msg.Payload = make([]byte, msg.Length)
	if _, err := io.ReadFull(port, msg.Payload); err != nil {
		return nil, err
	}

	// Read checksum
	checksumBytes := make([]byte, 2)
	if _, err := io.ReadFull(port, checksumBytes); err != nil {
		return nil, err
	}
	msg.Checksum = uint16(checksumBytes[0]) | uint16(checksumBytes[1])<<8

	return msg, nil
}

// ListSerialPorts lists available serial ports
func ListSerialPorts() ([]string, error) {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		return nil, err
	}

	var portNames []string
	for _, port := range ports {
		if port.IsUSB {
			portNames = append(portNames, port.Name)
		}
	}

	return portNames, nil
}
