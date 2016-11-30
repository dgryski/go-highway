// Package highway implements Google's HighwayHash
/*
   https://github.com/google/highwayhash
*/
package highway

import (
	"encoding/binary"
)

const (
	NumLanes   = 4
	packetSize = 8 * NumLanes
)

type Lanes [NumLanes]uint64

var (
	init0 = Lanes{0xdbe6d5d5fe4cce2f, 0xa4093822299f31d0, 0x13198a2e03707344, 0x243f6a8885a308d3}
	init1 = Lanes{0x3bd39e10cb0ef593, 0xc0acf169b5f18a8c, 0xbe5466cf34e90c6c, 0x452821e638d01377}
)

const useSSE = false

type state struct {
	v0, v1     Lanes
	mul0, mul1 Lanes
}

func newstate(s *state, keys Lanes) {
	var permutedKeys Lanes
	permute(&keys, &permutedKeys)
	for lane := range keys {
		s.v0[lane] = init0[lane] ^ keys[lane]
		s.v1[lane] = init1[lane] ^ permutedKeys[lane]
		s.mul0[lane] = init0[lane]
		s.mul1[lane] = init1[lane]
	}
}

func (s *state) Update(packet []byte) {
	for lane := 0; lane < NumLanes; lane++ {
		s.v1[lane] += binary.LittleEndian.Uint64(packet[8*lane:])
		s.v1[lane] += s.mul0[lane]
		const mask32 = 0xFFFFFFFF
		v0_32 := s.v0[lane] & mask32
		v1_32 := s.v1[lane] & mask32
		s.mul0[lane] ^= v0_32 * (s.v1[lane] >> 32)
		s.v0[lane] += s.mul1[lane]
		s.mul1[lane] ^= v1_32 * (s.v0[lane] >> 32)
	}

	var merged1 [32]byte
	zipperMerge(&s.v1, &merged1)
	for lane := range s.v0 {
		s.v0[lane] += binary.LittleEndian.Uint64(merged1[8*lane:])
	}

	var merged0 [32]byte
	zipperMerge(&s.v0, &merged0)
	for lane := range s.v1 {
		s.v1[lane] += binary.LittleEndian.Uint64(merged0[8*lane:])
	}
}

func (s *state) Finalize() uint64 {

	s.PermuteAndUpdate()
	s.PermuteAndUpdate()
	s.PermuteAndUpdate()
	s.PermuteAndUpdate()

	return s.v0[0] + s.v1[0] + s.mul0[0] + s.mul1[0]
}

func zipperMerge(mul0 *Lanes, v0 *[32]byte) {

	var mul0b [packetSize]byte
	binary.LittleEndian.PutUint64(mul0b[0:], mul0[0])
	binary.LittleEndian.PutUint64(mul0b[8:], mul0[1])
	binary.LittleEndian.PutUint64(mul0b[16:], mul0[2])
	binary.LittleEndian.PutUint64(mul0b[24:], mul0[3])

	for half := 0; half < packetSize; half += packetSize / 2 {
		v0[half+0] = mul0b[half+3]
		v0[half+1] = mul0b[half+12]
		v0[half+2] = mul0b[half+2]
		v0[half+3] = mul0b[half+5]
		v0[half+4] = mul0b[half+14]
		v0[half+5] = mul0b[half+1]
		v0[half+6] = mul0b[half+15]
		v0[half+7] = mul0b[half+0]
		v0[half+8] = mul0b[half+11]
		v0[half+9] = mul0b[half+4]
		v0[half+10] = mul0b[half+10]
		v0[half+11] = mul0b[half+13]
		v0[half+12] = mul0b[half+9]
		v0[half+13] = mul0b[half+6]
		v0[half+14] = mul0b[half+8]
		v0[half+15] = mul0b[half+7]
	}
}

func rot32(x uint64) uint64 {
	return (x >> 32) | (x << 32)
}

func permute(v, permuted *Lanes) {
	permuted[0] = rot32(v[2])
	permuted[1] = rot32(v[3])
	permuted[2] = rot32(v[0])
	permuted[3] = rot32(v[1])
}

func (s *state) PermuteAndUpdate() {
	var permuted Lanes

	permute(&s.v0, &permuted)

	var bytes [32]byte

	binary.LittleEndian.PutUint64(bytes[0:], permuted[0])
	binary.LittleEndian.PutUint64(bytes[8:], permuted[1])
	binary.LittleEndian.PutUint64(bytes[16:], permuted[2])
	binary.LittleEndian.PutUint64(bytes[24:], permuted[3])

	s.Update(bytes[:])
}

func Hash(key Lanes, bytes []byte) uint64 {

	if useSSE {
		return hashSSE(&key, &init0, &init1, bytes)
	}

	var s state

	size := len(bytes)
	remainder := size & (packetSize - 1)

	newstate(&s, key)
	// Hash entire 32-byte packets.
	truncatedSize := size - remainder
	for i := 0; i < truncatedSize/8; i += NumLanes {
		s.Update(bytes)
		bytes = bytes[32:]
	}

	// Update with final 32-byte packet.
	remainderMod4 := remainder & 3
	packet4 := uint32(size) << 24
	finalBytes := bytes[len(bytes)-remainderMod4:]
	for i := 0; i < remainderMod4; i++ {
		packet4 += uint32(finalBytes[i]) << uint(i*8)
	}

	var finalPacket [packetSize]byte
	copy(finalPacket[:], bytes[:len(bytes)-remainderMod4])
	binary.LittleEndian.PutUint32(finalPacket[packetSize-4:], packet4)

	s.Update(finalPacket[:])

	return s.Finalize()
}
