package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"math/big"
)

// HashStringToUint64 deterministically hashes a string identifier into a uint64
// using a truncated SHA-256 (first 8 bytes of the 32-byte sum) and
// binary.BigEndian.Uint64 to produce a secure, deterministic mapping.
// NOTE: This replaces the previous non-cryptographic FNV implementation and
// will change numeric outputs; coordinate with downstream systems if needed.
func HashStringToUint64(s string) uint64 {
	// Compute SHA-256 and take the first 8 bytes (big-endian) for uint64
	sum := sha256.Sum256([]byte(s))
	return binary.BigEndian.Uint64(sum[:8])
}

// TODO: Delete (depricated)
// SecureRandomInt64 generates a cryptographically secure random int64
// used primarily for creating Multi-Party Computation (MPC) secret shares.
// It relies on the operating system's entropy pool (crypto/rand) to ensure
// the unpredictability of the generated shares.
func SecureRandomInt64() int64 {
	// Limit the maximum number to 2^40 to prevent integer overflows
	// during summation in the MPC nodes.
	maxNum := big.NewInt(1 << 40)

	n, err := rand.Int(rand.Reader, maxNum)
	if err != nil {
		// Fallback to 0 in case of a critical OS entropy failure
		return 0
	}

	return n.Int64()
}

// SecureRandomUint64 generiše kriptografski siguran uint64
// oslanjajući se na cijeli Z_{2^64} opseg.
func SecureRandomUint64() uint64 {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return 0
	}
	return binary.LittleEndian.Uint64(b[:])
}
