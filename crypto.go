package wendy

import (
	"encoding/hex"
	"math/rand"
	"strings"
	"time"
)

// Rand is a random generator using `time.Now()` as the seed.
var Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()),
)

// Pubkey represents a public key bytes.
type Pubkey []byte

// NewPubkeyFromString turns a `[0x]<HEX>` (0x prefix is optional) string into
// a Pubkey.
// If the format of s is not hex encoded it panics.
func NewPubkeyFromID(id ID) Pubkey {
	s := string(id)
	s = strings.TrimPrefix(s, "0x")

	enc, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}

	return enc
}

// String returns the string representation of the public key. This is hex
// encoded and 0x prefixed.
func (pk Pubkey) String() string { return "0x" + hex.EncodeToString(pk) }
func (pk Pubkey) Bytes() []byte  { return pk }
