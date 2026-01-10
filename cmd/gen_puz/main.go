package main

import (
	"encoding/binary"
	"os"
)

func main() {
	f, err := os.Create("internal/app/testdata/sample.puz")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Header (0x34 bytes)
	binary.Write(f, binary.LittleEndian, uint16(0)) // Checksum
	f.Write([]byte("ACROSS&DOWN\x00"))              // Magic
	binary.Write(f, binary.LittleEndian, uint16(0)) // CIB Checksum
	f.Write(make([]byte, 8))                        // Masked Checksums
	f.Write([]byte("1.3\x00"))                      // Version
	binary.Write(f, binary.LittleEndian, uint16(0)) // Reserved
	binary.Write(f, binary.LittleEndian, uint16(0)) // Scrambled Checksum
	f.Write(make([]byte, 12))                       // Reserved
	f.Write([]byte{5, 5})                           // Width, Height
	binary.Write(f, binary.LittleEndian, uint16(8)) // Num Clues
	binary.Write(f, binary.LittleEndian, uint16(0)) // MaskBit
	binary.Write(f, binary.LittleEndian, uint16(0)) // Scrambled2

	// Solution (25 bytes)
	sol := "ABCDE" + "F.G.H" + "IJKLM" + "N.O.P" + "QRSTU"
	f.Write([]byte(sol))

	// State (25 bytes) - use '-' for empty
	state := "-----.---.-----.---.-----"
	f.Write([]byte(state))

	// Strings
	f.Write([]byte("Sample Title\x00"))
	f.Write([]byte("Sample Author\x00"))
	f.Write([]byte("Sample Copyright\x00"))
	// 8 clues
	for i := 1; i <= 8; i++ {
		f.Write([]byte("Clue text\x00"))
	}
	f.Write([]byte("Notes\x00"))
}
