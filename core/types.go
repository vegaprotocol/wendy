package core

import (
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
}

type Validator string

type Vote struct {
	Pubkey ID
	// TODO: signature

	Seq    uint64
	TxHash Hash
	Time   time.Time
}

func newVote(pub ID, seq uint64, tx Tx) *Vote {
	return &Vote{Pubkey: pub, Seq: seq, TxHash: tx.Hash(), Time: time.Now()}
}

func (v *Vote) String() string {
	return fmt.Sprintf("<pubkey=%s seq=%d, hash=%s>", v.Pubkey, v.Seq, v.TxHash)
}
