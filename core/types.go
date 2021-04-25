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
	Seq    uint64
	TxHash Hash
	Time   time.Time
	// TODO: signature
}

func newVote(seq uint64, tx Tx) *Vote {
	return &Vote{Seq: seq, TxHash: tx.Hash(), Time: time.Now()}
}

func (v *Vote) String() string {
	return fmt.Sprintf("<seq=%d, hash=%s>", v.Seq, v.TxHash)
}
