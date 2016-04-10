package highway

import "testing"

var total uint64
var buf = make([]byte, 8<<10)

func BenchmarkHighway8(b *testing.B)  { benchmarkHash(b, 8) }
func BenchmarkHighway16(b *testing.B) { benchmarkHash(b, 16) }
func BenchmarkHighway40(b *testing.B) { benchmarkHash(b, 40) }
func BenchmarkHighway64(b *testing.B) { benchmarkHash(b, 64) }
func BenchmarkHighway1K(b *testing.B) { benchmarkHash(b, 1024) }
func BenchmarkHighway8K(b *testing.B) { benchmarkHash(b, 8192) }

func benchmarkHash(b *testing.B, size int64) {
	b.SetBytes(size)
	bsz := buf[:size]
	total = 0
	keys := Lanes{}
	for i := 0; i < b.N; i++ {
		total += Hash(keys, bsz)
	}
}
