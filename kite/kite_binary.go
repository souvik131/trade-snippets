package kite

import (
	"encoding/binary"
)

type Packet []byte

func (packet Packet) ParseBinary(maxSize int) []uint32 {

	values := []uint32{}
	for {
		if len(packet) < 4 || len(values) >= maxSize/4 {
			return values
		}
		value := binary.BigEndian.Uint32(packet[0:4])
		values = append(values, value)
		if len(packet) > 4 {
			packet = packet[4:]
		}
	}

}

func (packet Packet) ParseMarketDepth() []uint32 {

	packetSize := len(packet)
	values := []uint32{}
	count := 0
	for {
		count++
		if len(packet) < 4 || len(values) >= packetSize/4 {
			return values
		}

		value := binary.BigEndian.Uint32(packet[0:4])
		if count%3 == 0 {
			value = uint32(binary.BigEndian.Uint16(packet[0:4]))
		}
		values = append(values, value)
		if len(packet) > 4 {
			packet = packet[4:]
		}
	}
}
