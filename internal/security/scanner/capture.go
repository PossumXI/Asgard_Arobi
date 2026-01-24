// Package scanner provides real network packet capture and analysis.
package scanner

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

// PacketCapture provides real packet capture from network interfaces.
type PacketCapture struct {
	handle     *pcap.Handle
	packetChan chan gopacket.Packet
	ctx        context.Context
	cancel     context.CancelFunc
	stats      Statistics
	mu         sync.RWMutex
}

// NewPacketCapture creates a new packet capture instance.
func NewPacketCapture(interfaceName string, snaplen int32, promiscuous bool) (*PacketCapture, error) {
	handle, err := pcap.OpenLive(interfaceName, snaplen, promiscuous, pcap.BlockForever)
	if err != nil {
		return nil, fmt.Errorf("failed to open interface %s: %w", interfaceName, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PacketCapture{
		handle:     handle,
		packetChan: make(chan gopacket.Packet, 1000),
		ctx:        ctx,
		cancel:     cancel,
		stats: Statistics{
			StartTime: time.Now(),
		},
	}, nil
}

// Start begins capturing packets.
func (pc *PacketCapture) Start(ctx context.Context) error {
	packetSource := gopacket.NewPacketSource(pc.handle, pc.handle.LinkType())

	go func() {
		defer close(pc.packetChan)
		for {
			select {
			case <-ctx.Done():
				return
			case <-pc.ctx.Done():
				return
			case packet := <-packetSource.Packets():
				if packet == nil {
					return
				}
				select {
				case pc.packetChan <- packet:
				case <-ctx.Done():
					return
				case <-pc.ctx.Done():
					return
				}
			}
		}
	}()

	return nil
}

// Stop stops packet capture.
func (pc *PacketCapture) Stop() error {
	pc.cancel()
	if pc.handle != nil {
		pc.handle.Close()
	}
	return nil
}

// GetPacketChannel returns the channel for captured packets.
func (pc *PacketCapture) GetPacketChannel() <-chan gopacket.Packet {
	return pc.packetChan
}

// ConvertPacket converts a gopacket.Packet to PacketInfo.
func (pc *PacketCapture) ConvertPacket(packet gopacket.Packet) PacketInfo {
	info := PacketInfo{
		Timestamp: packet.Metadata().Timestamp,
		Size:      len(packet.Data()),
	}

	// Extract IP layer
	if ipLayer := packet.Layer(layers.LayerTypeIPv4); ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		info.SourceIP = ip.SrcIP
		info.DestIP = ip.DstIP
		info.Protocol = ip.Protocol.String()
	}

	// Extract TCP layer
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		info.SourcePort = int(tcp.SrcPort)
		info.DestPort = int(tcp.DstPort)
		info.Flags = tcpFlagsToString(tcp)
		info.Payload = tcp.Payload
	}

	// Extract UDP layer
	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		info.SourcePort = int(udp.SrcPort)
		info.DestPort = int(udp.DstPort)
		info.Protocol = "UDP"
		info.Payload = udp.Payload
	}

	return info
}

func tcpFlagsToString(tcp *layers.TCP) string {
	var flags []string
	if tcp.FIN {
		flags = append(flags, "FIN")
	}
	if tcp.SYN {
		flags = append(flags, "SYN")
	}
	if tcp.RST {
		flags = append(flags, "RST")
	}
	if tcp.PSH {
		flags = append(flags, "PSH")
	}
	if tcp.ACK {
		flags = append(flags, "ACK")
	}
	if tcp.URG {
		flags = append(flags, "URG")
	}
	if len(flags) == 0 {
		return "NONE"
	}
	return fmt.Sprintf("%v", flags)
}
