package highway

import (
	"reflect"
	"testing"
)

func TestHighway(t *testing.T) {

	input := make([]byte, 128)

	var tests = []uint64{
		0xe59e60a55ba25cca, 0x0eaff68bcbebbfb8, 0x26c78b9cd72d6e48, 0x6b2a247687f60d46,
		0xdf5d85049306e048, 0xe1984edd081ee203, 0xaad8085e7d5b5b05, 0x973ded9774fdbf6d,
		0xeb899858319e72f6, 0xfc20f51850bc1b1d, 0x68259661bc0c2c94, 0xe571ee0943190caf,
		0x9f9cfad6f461c002, 0x6f134ac1d5fc262e, 0x794d424c6f7f47af, 0xf0b946d1589c1ce1,
		0xaaf71d3c0a6b8222, 0xa2ffca61066fa7f0, 0x775ed502565c721d, 0x5d473d91b598f573,
		0x9e8f30af520e8789, 0x48f180ec3f007b4c, 0x5cf2bb7fdd878236, 0xde7794eee547f636,
		0x728e0902c9018a47, 0xf13a72a46b9cbfc2, 0x3578d814af754b8c, 0x84ecf3737bf8c7d8,
		0x6813af6f5cd2a6e8, 0xcf1715ab49b98c62, 0x292fce041f8e69b1, 0xd9e1d4354a6251e2,
		0xb49e53a4ccb6530b, 0x4d4ad2f6bee8b90e, 0xfc13391dc6a5d5e5, 0xfd2ffdaf1d74b1c4,
		0xdf26439b7b3d7881, 0x7c7aba402e0cb201, 0x33c43024f95c23db, 0xfbf644033410a627,
		0x93ce500e04a5c20b, 0xf8385df994fc5a31, 0x1600988ae780b4f5, 0x3bd796e032319b42,
		0xa2cdc5fc9e43f2e2, 0x83f2824b2a0ae8f6, 0xfe891f9491dcaf48, 0x13f4fe6763b6b4be,
		0x37f585d3ec1df839, 0x15215fd181147cea, 0xdbc403171b9a1812, 0xf58d228800af9f1a,
		0x596480df630da49e, 0x411971159edc16d8, 0x5052ef7c553746f7, 0xee06d8268611305e,
		0x057ee62fbd62a3dc, 0xd57feb47b862cd88, 0x8f56ffe7bbfb0342, 0xecfbfb8047a2d248,
		0xd6e69648885b4359, 0x3bd5ec097cdbe245, 0x3bda6558708dde73, 0x250be1ccd36ebcb7,
		0x9c7054dbf83ad0c6, 0x4868ffcbb561a2c8, 0x3130b9e44b7a4fe1, 0x02e3be039c26f22b,
		0xd8594a6fa2d56b5e, 0x83c9e9ec6b1a854b, 0xafdfaedb70ffff1f, 0x9fdf3d030ad9c934,
		0x808a870f8ffbeaff, 0xa0e686233ea5fd6c, 0x2d71790f49f18cf0, 0xc9eefdd0f9adbda7,
		0xdf5c0b1075954409, 0xf92fb4b370d3b852, 0x70c2768648373c67, 0x8a3e1bd634a519bd,
		0x4002360135bc912b, 0xdbfa60279e411167, 0xc6f0b6dfda74e402, 0x52965f777e34b2ea,
		0x48a8ec7c5c488313, 0x19a5d3e70bfc6b14, 0xd9e6fd39f53fc5d6, 0x8b9af7d8ef4175d9,
		0x0caea2ea592dd67c, 0x26a9ef6c8e385790, 0x24ae78455d0a513c, 0xa6f5b9c6a68bd6da,
		0x690f9c60f277822c, 0xc4714745d3163fc4, 0x72a79062b09309bd, 0x3e508e993cd70a97,
		0x6b1f9e6e2d5ce838, 0xf26eb25550990da5, 0x90dd81251eeb49c7, 0xaa5fb4cf553745c8,
		0x9968a7b91ed27e41, 0x0c9c97c698d1134d, 0xc8e40f397c879392, 0xcd5735d13f0c1fd5,
		0xa5257de6529d81f6, 0x94470b0486a10c19, 0xbd3ab11e6e8c0e30, 0xe57bdff3b94cab69,
		0xe7d7784a1f2e6fd3, 0xc4e7e51d065e9be5, 0x518281813d93944d, 0x1732f7680ff98061,
		0xc26efbbce597d771, 0x1074b08577f20d3a, 0x7c302611d95504aa, 0x347cae275842b5eb,
		0x26f4514787a7eccb, 0x406ea37cc5414037, 0x8aed501ca9d2c3b0, 0x959976fa5fe2484f,
		0x88af21f06aebc900, 0xa93f2ac244aa4650, 0xd42744021372b343, 0x2e55f90b119cc7be,
		0xaed0cfa3648896c2, 0xe237ee374fb467a6, 0x64e153eca3d22d5c, 0x6975aac6e07837ed,
	}

	key := Lanes{0x0706050403020100, 0x0F0E0D0C0B0A0908, 0x1716151413121110, 0x1F1E1D1C1B1A1918}

	for i := range input {
		input[i] = byte(i)

		if h := Hash(key, input[:i]); h != tests[i] {
			t.Errorf("Hash(..., input[:%d])=%016x, want %016x\n", i, h, tests[i])
		}
	}
}

func TestPermuteSSE(t *testing.T) {
	v1 := Lanes{0x0001020304050607, 0x08090A0B0C0D0E0F, 0x1011121314151617, 0x18191A1B1C1D1E1F}
	var p Lanes

	permute(&v1, &p)

	var psse Lanes

	permuteSSE(&v1, &psse)

	if !reflect.DeepEqual(p, psse) {
		t.Errorf("permuteSSE")
		t.Logf("got : %x", psse)
		t.Logf("want: %x", p)
	}
}

func TestZipperSSE(t *testing.T) {
	v1 := Lanes{0x0001020304050607, 0x08090A0B0C0D0E0F, 0x1011121314151617, 0x18191A1B1C1D1E1F}
	var p [32]byte

	zipperMerge(&v1, &p)

	var psse [32]byte

	zipperSSE(&v1, &psse)

	if !reflect.DeepEqual(p, psse) {
		t.Errorf("zipperSSE")
		t.Logf("got : %x", psse)
		t.Logf("want: %x", p)
	}
}

func TestUpdateSSE(t *testing.T) {

	s := newstate(Lanes{})
	var p [32]byte

	s.Update(p[:])

	s2 := newstate(Lanes{})

	updateSSE(&s2, p[:])

	if !reflect.DeepEqual(s, s2) {
		t.Errorf("updateSSE")
		t.Logf("got : %x", s2)
		t.Logf("want: %x", s)
	}
}

var sink uint64

func BenchmarkPermute(b *testing.B) {

	v1 := Lanes{0x0001020304050607, 0x08090A0B0C0D0E0F, 0x1011121314151617, 0x18191A1B1C1D1E1F}
	var p Lanes

	for i := 0; i < b.N; i++ {
		permute(&v1, &p)
	}

	sink += p[0]
}

func BenchmarkPermuteSSE(b *testing.B) {

	v1 := Lanes{0x0001020304050607, 0x08090A0B0C0D0E0F, 0x1011121314151617, 0x18191A1B1C1D1E1F}
	var p Lanes

	for i := 0; i < b.N; i++ {
		permuteSSE(&v1, &p)
	}

	sink += p[0]
}

func BenchmarkZipper(b *testing.B) {

	v1 := Lanes{0x0001020304050607, 0x08090A0B0C0D0E0F, 0x1011121314151617, 0x18191A1B1C1D1E1F}
	var p [32]byte

	for i := 0; i < b.N; i++ {
		zipperMerge(&v1, &p)
	}

	sink += uint64(p[0])
}

func BenchmarkZipperSSE(b *testing.B) {

	v1 := Lanes{0x0001020304050607, 0x08090A0B0C0D0E0F, 0x1011121314151617, 0x18191A1B1C1D1E1F}
	var p [32]byte

	for i := 0; i < b.N; i++ {
		zipperSSE(&v1, &p)
	}

	sink += uint64(p[0])
}
