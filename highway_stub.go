package highway

//go:generate python -m peachpy.x86_64 sum.py -S -o highway_amd64.s -mabi=goasm
//go:noescape

func hashSSE(keys, init0, init1 *Lanes, p []byte) uint64
