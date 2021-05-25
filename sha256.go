package wendy

import (
	"crypto/sha256"
)

// Checksum returns a sha256 checksum of v
func Checksum(v []byte) Hash {
	hash := sha256.Sum256(v)
	return Hash(hash)
}
