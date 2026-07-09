package varcore

import (
	"fmt"
	"unicode/utf16"
)

// FNV-1a (32-bit) constants.
const (
	fnvOffset uint32 = 0x811c9dc5
	fnvPrime  uint32 = 0x01000193
)

// HashSource is a FNV-1a (32-bit) change-detector over the UTF-16 code units of
// source. Port of hash.ts / hash.py. Not a security hash: tiny,
// dependency-free, and byte-identical across every port so var.lock.json
// fingerprints match. The "fnv1a:" prefix namespaces the algorithm.
//
// Go strings are UTF-8, so the runes are re-encoded to UTF-16 code units first
// (an astral character contributes two units, matching JS string semantics).
func HashSource(source string) string {
	h := fnvOffset
	for _, unit := range utf16.Encode([]rune(source)) {
		h ^= uint32(unit)
		h *= fnvPrime
	}
	return fmt.Sprintf("fnv1a:%08x", h)
}
