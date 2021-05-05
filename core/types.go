package core

import (
	"bytes"
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
