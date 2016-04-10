// Package highway implements Google's HighwayHash
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

type state struct {
	v0, v1 Lanes
}

func newstate(keys Lanes) state {
	var s state
	for lane, key := range keys {
		s.v0[lane] = init0[lane] ^ key
		s.v1[lane] = init1[lane] ^ key
	}

	return s
}

func (s *state) Update(packet []byte) {

	var mul1 Lanes

	var mul0 [32]byte

	for lane := 0; lane < NumLanes; lane++ {
		s.v1[lane] += binary.LittleEndian.Uint64(packet[8*lane:])
		const mask32 = 0xFFFFFFFF
		s.v0[lane] |= 0x70000001
		mul32 := s.v0[lane] & mask32
		binary.LittleEndian.PutUint64(mul0[8*lane:], mul32*(s.v1[lane]&mask32))
		mul1[lane] = mul32 * (s.v1[lane] >> 32)
	}

	var merged [32]byte
	s.ZipperMerge(mul0[:], merged[:])

	for lane := range mul1 {
		s.v0[lane] += binary.LittleEndian.Uint64(merged[8*lane:])
		s.v1[lane] += mul1[lane]
	}
}

func (s *state) Finalize() uint64 {

	s.PermuteAndUpdate()
	s.PermuteAndUpdate()
	s.PermuteAndUpdate()
	s.PermuteAndUpdate()

	return s.v0[0] + s.v1[0]
}

func (s *state) ZipperMerge(mul0, v0 []byte) []byte {

	for half := 0; half < packetSize; half += packetSize / 2 {
		v0[half+0] = mul0[half+3]
		v0[half+1] = mul0[half+12]
		v0[half+2] = mul0[half+2]
		v0[half+3] = mul0[half+5]
		v0[half+4] = mul0[half+14]
		v0[half+5] = mul0[half+1]
		v0[half+6] = mul0[half+15]
		v0[half+7] = mul0[half+0]
		v0[half+8] = mul0[half+11]
		v0[half+9] = mul0[half+4]
		v0[half+10] = mul0[half+10]
		v0[half+11] = mul0[half+13]
		v0[half+12] = mul0[half+9]
		v0[half+13] = mul0[half+6]
		v0[half+14] = mul0[half+8]
		v0[half+15] = mul0[half+7]
	}

	return v0
}

func rot32(x uint64) uint64 {
	return (x >> 32) | (x << 32)
}

func (s *state) PermuteAndUpdate() {
	var permuted [32]byte

	binary.LittleEndian.PutUint64(permuted[0:], rot32(s.v0[2]))
	binary.LittleEndian.PutUint64(permuted[8:], rot32(s.v0[3]))
	binary.LittleEndian.PutUint64(permuted[16:], rot32(s.v0[0]))
	binary.LittleEndian.PutUint64(permuted[24:], rot32(s.v0[1]))

	s.Update(permuted[:])
}

func Hash(key Lanes, bytes []byte) uint64 {

	s := newstate(key)

	size := len(bytes)

	// Hash entire 32-byte packets.
	remainder := size & (packetSize - 1)
	truncatedSize := size - remainder
	// var packets []uint64 // reinterpret_cast<const uint64_t*>(bytes);
	biter := bytes
	for i := 0; i < truncatedSize/8; i += NumLanes {
		s.Update(biter)
		biter = biter[32:]
	}

	// Update with final 32-byte packet.
	remainderMod4 := remainder & 3
	packet4 := uint32(size) << 24
	finalBytes := bytes[size-remainderMod4:]
	for i := 0; i < remainderMod4; i++ {
		packet4 += uint32(finalBytes[i]) << uint(i*8)
	}

	var finalPacket [packetSize]byte
	copy(finalPacket[:], bytes[truncatedSize:size-remainderMod4])
	binary.LittleEndian.PutUint32(finalPacket[packetSize-4:], packet4)

	s.Update(finalPacket[:])

	return s.Finalize()
}
