package wendy

import (
	"bytes"
	"crypto/ed25519"
	"encoding/binary"
	"fmt"
	"time"
)

const (
	HashLen = 32
)

var (
	Quorum = float64(2) / 3
)

type Hash [HashLen]byte

func (h Hash) String() string { return string(h[:]) }

type ID string

type Tx interface {
	Bytes() []byte
	Hash() Hash
	Label() string
}

type BlockingSet map[Hash][]Tx

func (set BlockingSet) String() string {
	var buf = &bytes.Buffer{}
	for hash, txs := range set {
		fmt.Fprintf(buf, "tx: %s, depends_on: %v\n", hash, txs)
	}
	return buf.String()
}

// Block holds a list of Tx.
type Block struct {
	Txs []Tx
}

type Validator []byte

type Vote struct {
	Pubkey Pubkey

	Label string

	// the following fields are used to produce the digest
	Seq      uint64
	TxHash   Hash
	Time     time.Time
	PrevHash Hash
}

// NewVote returns a new Vote
func NewVote(pub Pubkey, seq uint64, tx Tx) *Vote {
	return &Vote{Pubkey: pub, Seq: seq, TxHash: tx.Hash(),
		Label: tx.Label(), Time: time.Now()}
}

func (v *Vote) String() string {
	return fmt.Sprintf("<pubkey=%s seq=%d, tx_hash=%s>", v.Pubkey, v.Seq, v.TxHash)
}

// WithPrevHash returns an updated Vote with hash set as .PrevHash.
func (v *Vote) WithPrevHash(hash Hash) *Vote {
	v.PrevHash = hash
	return v
}

func (v *Vote) digest() []byte {
	buf := bytes.NewBuffer(nil)

	// the following are the fields used to produce the digest.
	for _, i := range []interface{}{
		v.Seq,
		v.TxHash,
		v.Time.UnixNano(),
		v.PrevHash,
	} {
		// according to buf.Buffer docs, it will never return an error
		// we should catch this in case of a change in their promise.
		if err := binary.Write(buf, binary.BigEndian, i); err != nil {
			panic(err)
		}
	}

	return buf.Bytes()
}

// Hash returns the sha256 hash of the vote's digest, which is the same digest
// used for signing.
func (v *Vote) Hash() Hash {
	return Checksum(v.digest())
}

func (v *Vote) Key() ID {
	return ID(v.Pubkey.String())
}

// SignedVote wraps a vote with its signature.
type SignedVote struct {
	Signature []byte
	Data      *Vote
}

// NewSignedVote signs a vote and return it wrapped inside a SignedVote.
func NewSignedVote(key ed25519.PrivateKey, v *Vote) *SignedVote {
	return &SignedVote{
		Signature: ed25519.Sign(key, v.digest()),
		Data:      v,
	}
}

// Verify verifies the signature from SignedVote given the vote's pubkey.
func (sv *SignedVote) Verify() bool {
	pub := ed25519.PublicKey(sv.Data.Pubkey)
	return ed25519.Verify(pub, sv.Data.digest(), sv.Signature)
}
