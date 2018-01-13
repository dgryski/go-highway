package highway

import (
	"testing"
)

func TestHighway(t *testing.T) {

	input := make([]byte, 64)

	var tests = []uint64{
		0x907a56de22c26e53, 0x7eab43aac7cddd78, 0xb8d0569ab0b53d62,
		0x5c6befab8a463d80, 0xf205a46893007eda, 0x2b8a1668e4a94541,
		0xbd4ccc325befca6f, 0x4d02ae1738f59482, 0xe1205108e55f3171,
		0x32d2644ec77a1584, 0xf6e10acdb103a90b, 0xc3bbf4615b415c15,
		0x243cc2040063fa9c, 0xa89a58ce65e641ff, 0x24b031a348455a23,
		0x40793f86a449f33b, 0xcfab3489f97eb832, 0x19fe67d2c8c5c0e2,
		0x04dd90a69c565cc2, 0x75d9518e2371c504, 0x38ad9b1141d3dd16,
		0x0264432ccd8a70e0, 0xa9db5a6288683390, 0xd7b05492003f028c,
		0x205f615aea59e51e, 0xeee0c89621052884, 0x1bfc1a93a7284f4f,
		0x512175b5b70da91d, 0xf71f8976a0a2c639, 0xae093fef1f84e3e7,
		0x22ca92b01161860f, 0x9fc7007ccf035a68, 0xa0c964d9ecd580fc,
		0x2c90f73ca03181fc, 0x185cf84e5691eb9e, 0x4fc1f5ef2752aa9b,
		0xf5b7391a5e0a33eb, 0xb9b84b83b4e96c9c, 0x5e42fe712a5cd9b4,
		0xa150f2f90c3f97dc, 0x7fa522d75e2d637d, 0x181ad0cc0dffd32b,
		0x3889ed981e854028, 0xfb4297e8c586ee2d, 0x6d064a45bb28059c,
		0x90563609b3ec860c, 0x7aa4fce94097c666, 0x1326bac06b911e08,
		0xb926168d2b154f34, 0x9919848945b1948d, 0xa2a98fc534825ebe,
		0xe9809095213ef0b6, 0x582e5483707bc0e9, 0x086e9414a88a6af5,
		0xee86b98d20f6743d, 0xf89b7ff609b1c0a7, 0x4c7d9cc19e22c3e8,
		0x9a97005024562a6f, 0x5dd41cf423e6ebef, 0xdf13609c0468e227,
		0x6e0da4f64188155a, 0xb755ba4b50d7d4a1, 0x887a3484647479bd,
		0xab8eebe9bf2139a0, 0x75542c5d4cd2a6ff,
	}

	key := Lanes{0x0706050403020100, 0x0F0E0D0C0B0A0908, 0x1716151413121110, 0x1F1E1D1C1B1A1918}

	for i := range input {
		input[i] = byte(i)

		if h := Hash(key, input[:i]); h != tests[i] {
			t.Errorf("Hash(..., input[:%d])=%016x, want %016x\n", i, h, tests[i])
		} else {
			t.Logf("PASS: Hash(..., input[:%d])=%016x, want %016x\n", i, h, tests[i])
		}
	}
}

func TestCompare(t *testing.T) {

	input := make([]byte, 64)

	key := Lanes{0x0706050403020100, 0x0F0E0D0C0B0A0908, 0x1716151413121110, 0x1F1E1D1C1B1A1918}

	for i := range input {
		input[i] = byte(i)

		want := Hash(key, input[:i])
		got := hashSSE(&key, &init0, &init1, input[:i])

		if got != want {
			t.Errorf("hashSSE(..., input[:%d])=%016x, want %016x\n", i, got, want)
		} else {
			t.Logf("PASS: hashSSE(..., input[:%d])=%016x\n", i, got)
		}
	}
}
