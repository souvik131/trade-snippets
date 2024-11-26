package kite

import (
	"encoding/binary"
)

type Packet []byte

// parseBinary parses the main packet data
func (packet Packet) parseBinary(maxSize int) []uint32 {
	if len(packet) < 4 {
		// log.Printf("Packet too short for even token: %d bytes", len(packet))
		return nil
	}

	// For LTP mode, we only need first 8 bytes
	if maxSize == 8 {
		if len(packet) < 8 {
			// log.Printf("LTP packet too short: %d bytes", len(packet))
			return nil
		}
		values := make([]uint32, 2)
		values[0] = binary.BigEndian.Uint32(packet[0:4]) // Token
		values[1] = binary.BigEndian.Uint32(packet[4:8]) // LTP
		// log.Printf("LTP packet parsed: token=%d, ltp=%d", values[0], values[1])
		return values
	}

	// For quote mode, we need 44 bytes
	if maxSize == 44 {
		if len(packet) < 44 {
			// log.Printf("Quote packet too short: %d bytes", len(packet))
			return nil
		}
		values := make([]uint32, 8)
		values[0] = binary.BigEndian.Uint32(packet[0:4])   // Token
		values[1] = binary.BigEndian.Uint32(packet[4:8])   // LTP
		values[2] = binary.BigEndian.Uint32(packet[32:36]) // High
		values[3] = binary.BigEndian.Uint32(packet[36:40]) // Low
		values[4] = binary.BigEndian.Uint32(packet[28:32]) // Open
		values[5] = binary.BigEndian.Uint32(packet[40:44]) // Close
		values[6] = binary.BigEndian.Uint32(packet[24:28]) // Price Change
		values[7] = binary.BigEndian.Uint32(packet[60:64]) // Exchange Timestamp
		// log.Printf("Quote packet parsed: token=%d, ltp=%d, high=%d, low=%d, open=%d, close=%d", values[0], values[1], values[2], values[3], values[4], values[5])
		return values
	}

	// For full mode, parse first 64 bytes
	if maxSize == 64 {
		if len(packet) < 64 {
			// log.Printf("Full packet too short: %d bytes", len(packet))
			return nil
		}
		values := make([]uint32, 16)
		values[0] = binary.BigEndian.Uint32(packet[0:4])    // Token
		values[1] = binary.BigEndian.Uint32(packet[4:8])    // LTP
		values[2] = binary.BigEndian.Uint32(packet[8:12])   // LTQ
		values[3] = binary.BigEndian.Uint32(packet[12:16])  // ATP
		values[4] = binary.BigEndian.Uint32(packet[16:20])  // Volume
		values[5] = binary.BigEndian.Uint32(packet[20:24])  // Buy Quantity
		values[6] = binary.BigEndian.Uint32(packet[24:28])  // Sell Quantity
		values[7] = binary.BigEndian.Uint32(packet[32:36])  // High
		values[8] = binary.BigEndian.Uint32(packet[36:40])  // Low
		values[9] = binary.BigEndian.Uint32(packet[28:32])  // Open
		values[10] = binary.BigEndian.Uint32(packet[40:44]) // Close
		values[11] = binary.BigEndian.Uint32(packet[44:48]) // Last Trade Time
		values[12] = binary.BigEndian.Uint32(packet[48:52]) // OI
		values[13] = binary.BigEndian.Uint32(packet[52:56]) // OI High
		values[14] = binary.BigEndian.Uint32(packet[56:60]) // OI Low
		values[15] = binary.BigEndian.Uint32(packet[60:64]) // Exchange Time
		// log.Printf("Full packet parsed: token=%d, ltp=%d, ltq=%d, volume=%d", values[0], values[1], values[2], values[4])
		return values
	}

	// log.Printf("Unknown packet size requested: %d", maxSize)
	return nil
}

// parseMarketDepth parses the market depth data
func (packet Packet) parseMarketDepth() []uint32 {
	if len(packet) < 12 { // Need at least one complete depth entry
		// log.Printf("Market depth packet too short: %d bytes", len(packet))
		return nil
	}

	// Each depth entry is 12 bytes (4 bytes each for quantity, price, orders)
	numEntries := len(packet) / 12
	// log.Printf("Parsing market depth with %d entries", numEntries)

	values := make([]uint32, numEntries*3)
	for i := 0; i < numEntries; i++ {
		offset := i * 12
		if offset+12 > len(packet) {
			// log.Printf("Incomplete depth entry at offset %d", offset)
			break
		}

		values[i*3] = binary.BigEndian.Uint32(packet[offset : offset+4])      // Quantity
		values[i*3+1] = binary.BigEndian.Uint32(packet[offset+4 : offset+8])  // Price
		values[i*3+2] = binary.BigEndian.Uint32(packet[offset+8 : offset+12]) // Orders

		// log.Printf("Depth entry %d: qty=%d, price=%d, orders=%d", i, values[i*3], values[i*3+1], values[i*3+2])
	}

	return values
}
