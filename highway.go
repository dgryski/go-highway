// Package highway implements Google's HighwayHash
package highway

import (
	"encoding/binary"
)

const (
	NumLanes   = 4
	PacketSize = 8 * NumLanes
)

type Lanes [NumLanes]uint64

var (
	init0 = Lanes{0xdbe6d5d5fe4cce2f, 0xa4093822299f31d0, 0x13198a2e03707344, 0x243f6a8885a308d3}
	init1 = Lanes{0x3bd39e10cb0ef593, 0xc0acf169b5f18a8c, 0xbe5466cf34e90c6c, 0x452821e638d01377}
)

const debug = false

type State struct {
	v0, v1 Lanes
}

func New(keys Lanes) *State {
	var s State
	for lane, key := range keys {
		s.v0[lane] = init0[lane] ^ key
		s.v1[lane] = init1[lane] ^ key
	}

	return &s
}

func (s *State) Update(packet *Lanes) {

	var mul0, mul1 Lanes

	for lane := range packet {
		s.v1[lane] += packet[lane]
		const mask32 = 0xFFFFFFFF
		s.v0[lane] |= 0x70000001
		mul32 := s.v0[lane] & mask32
		mul0[lane] = mul32 * (s.v1[lane] & mask32)
		mul1[lane] = mul32 * (s.v1[lane] >> 32)
	}

	merged := s.ZipperMerge(&mul0)
	for lane := range merged {
		s.v0[lane] += merged[lane]
		s.v1[lane] += mul1[lane]
	}
}

func (s *State) Finalize() uint64 {

	s.PermuteAndUpdate()
	s.PermuteAndUpdate()
	s.PermuteAndUpdate()
	s.PermuteAndUpdate()

	return s.v0[0] + s.v1[0]
}

func (s *State) ZipperMerge(mul0 *Lanes) Lanes {

	var mul0b [PacketSize]byte
	binary.LittleEndian.PutUint64(mul0b[0:], mul0[0])
	binary.LittleEndian.PutUint64(mul0b[8:], mul0[1])
	binary.LittleEndian.PutUint64(mul0b[16:], mul0[2])
	binary.LittleEndian.PutUint64(mul0b[24:], mul0[3])

	var v0 [PacketSize]byte

	for half := 0; half < PacketSize; half += PacketSize / 2 {
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

	return Lanes{
		binary.LittleEndian.Uint64(v0[0:]),
		binary.LittleEndian.Uint64(v0[8:]),
		binary.LittleEndian.Uint64(v0[16:]),
		binary.LittleEndian.Uint64(v0[24:]),
	}
}

func Rot32(x uint64) uint64 {
	return (x >> 32) | (x << 32)
}

func (s *State) PermuteAndUpdate() {
	permuted := Lanes{Rot32(s.v0[2]), Rot32(s.v0[3]), Rot32(s.v0[0]), Rot32(s.v0[1])}
	s.Update(&permuted)
}

func Hash(key Lanes, bytes []byte) uint64 {

	s := New(key)

	size := len(bytes)

	// Hash entire 32-byte packets.
	remainder := size & (PacketSize - 1)
	truncated_size := size - remainder
	// var packets []uint64 // reinterpret_cast<const uint64_t*>(bytes);
	biter := bytes
	for i := 0; i < truncated_size/8; i += NumLanes {
		var packet = Lanes{
			binary.LittleEndian.Uint64(biter[0:]),
			binary.LittleEndian.Uint64(biter[8:]),
			binary.LittleEndian.Uint64(biter[16:]),
			binary.LittleEndian.Uint64(biter[24:]),
		}
		biter = biter[32:]

		s.Update(&packet)
	}

	// Update with final 32-byte packet.
	remainder_mod4 := remainder & 3
	packet4 := uint32(size) << 24
	final_bytes := bytes[size-remainder_mod4:]
	for i := 0; i < remainder_mod4; i++ {
		packet4 += uint32(final_bytes[i]) << uint(i*8)
	}

	var final_packet [PacketSize]byte
	copy(final_packet[:], bytes[truncated_size:size-remainder_mod4])
	binary.LittleEndian.PutUint32(final_packet[PacketSize-4:], packet4)

	var packet = Lanes{
		binary.LittleEndian.Uint64(final_packet[0:]),
		binary.LittleEndian.Uint64(final_packet[8:]),
		binary.LittleEndian.Uint64(final_packet[16:]),
		binary.LittleEndian.Uint64(final_packet[24:]),
	}

	s.Update(&packet)

	return s.Finalize()
}
