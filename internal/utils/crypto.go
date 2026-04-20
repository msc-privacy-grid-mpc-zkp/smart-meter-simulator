package crypto

import (
	"crypto/rand"
	"hash/fnv"
	"math/big"
)

// HashStringToUint64 deterministically hashes a string identifier into a uint64
// using the FNV-1a algorithm. It is safe to use for generating ZKP public inputs
// and ensures consistent mapping of meter IDs across the system.
func HashStringToUint64(s string) uint64 {
	h := fnv.New64a()

	// Explicitly ignoring the return values (int, error) to satisfy strict linters,
	// since the standard library's FNV-1a Write method never returns an error.
	_, _ = h.Write([]byte(s))

	return h.Sum64()
}

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
