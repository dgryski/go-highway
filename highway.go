// Package highway implements Google's HighwayHash
/*
   https://github.com/google/highwayhash
*/
package highway

import (
	"encoding/binary"

	"github.com/intel-go/cpuid"
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

var useSSE = cpuid.HasFeature(cpuid.SSE4_1)

type state struct {
	v0, v1     Lanes
	mul0, mul1 Lanes
}

func newstate(s *state, keys Lanes) {
	var permutedKeys Lanes
	rotate64by32(&keys, &permutedKeys)
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
		v1_32 := s.v1[lane] & mask32
		s.mul0[lane] ^= v1_32 * (s.v0[lane] >> 32)
		s.v0[lane] += s.mul1[lane]
		v0_32 := s.v0[lane] & mask32
		s.mul1[lane] ^= v0_32 * (s.v1[lane] >> 32)
	}

	zipperMergeAndAdd(s.v1[1], s.v1[0], &s.v0[1], &s.v0[0])
	zipperMergeAndAdd(s.v1[3], s.v1[2], &s.v0[3], &s.v0[2])
	zipperMergeAndAdd(s.v0[1], s.v0[0], &s.v1[1], &s.v1[0])
	zipperMergeAndAdd(s.v0[3], s.v0[2], &s.v1[3], &s.v1[2])
}

func (s *state) Finalize() uint64 {

	s.PermuteAndUpdate()
	s.PermuteAndUpdate()
	s.PermuteAndUpdate()
	s.PermuteAndUpdate()

	return s.v0[0] + s.v1[0] + s.mul0[0] + s.mul1[0]
}

func zipperMergeAndAdd(v1, v0 uint64, add1, add0 *uint64) {
	*add0 += (((v0 & 0xff000000) | (v1 & 0xff00000000)) >> 24) |
		(((v0 & 0xff0000000000) | (v1 & 0xff000000000000)) >> 16) |
		(v0 & 0xff0000) | ((v0 & 0xff00) << 32) |
		((v1 & 0xff00000000000000) >> 8) | (v0 << 56)
	*add1 += (((v1 & 0xff000000) | (v0 & 0xff00000000)) >> 24) |
		(v1 & 0xff0000) | ((v1 & 0xff0000000000) >> 16) |
		((v1 & 0xff00) << 24) | ((v0 & 0xff000000000000) >> 8) |
		((v1 & 0xff) << 48) | (v0 & 0xff00000000000000)
}

func rot32(x uint64) uint64 {
	return (x >> 32) | (x << 32)
}

func rotate32By(count uint, lanes *Lanes) {
	for i := 0; i < 4; i++ {
		half0 := uint32(lanes[i] & 0xffffffff)
		half1 := uint32(lanes[i] >> 32)
		lanes[i] = uint64(half0<<count) | uint64(half0>>(32-count))
		lanes[i] |= uint64((half1<<count)|(half1>>(32-count))) << 32
	}
}

func rotate64by32(v, permuted *Lanes) {
	permuted[0] = rot32(v[0])
	permuted[1] = rot32(v[1])
	permuted[2] = rot32(v[2])
	permuted[3] = rot32(v[3])
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
	sizeMod32 := size & (packetSize - 1)

	newstate(&s, key)
	// Hash entire 32-byte packets.
	truncatedSize := size - sizeMod32
	for i := 0; i < truncatedSize/8; i += NumLanes {
		s.Update(bytes)
		bytes = bytes[32:]
	}

	if sizeMod32 != 0 {
		// Update with final 32-byte packet.
		for i := 0; i < NumLanes; i++ {
			s.v0[i] += uint64(sizeMod32)<<32 + uint64(sizeMod32)
		}
		rotate32By(uint(sizeMod32), &s.v1)

		sizeMod4 := sizeMod32 & 3
		var finalPacket [packetSize]byte
		copy(finalPacket[:], bytes[:len(bytes)-sizeMod4])
		remainder := bytes[len(bytes)-sizeMod4:]

		if sizeMod32&16 != 0 {
			copy(finalPacket[28:], bytes[len(bytes)-4:])
		} else {
			if sizeMod4 != 0 {
				finalPacket[16+0] = remainder[0]
				finalPacket[16+1] = remainder[sizeMod4>>1]
				finalPacket[16+2] = remainder[sizeMod4-1]
			}
		}

		s.Update(finalPacket[:])
	}

	return s.Finalize()
}
