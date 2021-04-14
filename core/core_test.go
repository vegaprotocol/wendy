package core

import (
	"testing"
	"time"
)

type testTx struct {
	bytes []byte
	hash  []byte
}

func (tx *testTx) Bytes() []byte { return tx.bytes }
func (tx *testTx) Hash() Hash {
	var hash Hash
	copy(hash[:], tx.hash)
	return hash
}

func TestVoting(t *testing.T) {
	w := New()

	tx := &testTx{}
	w.AddTx(tx)

	vote := &Vote{Seq: 0, TxHash: tx.Hash(), Time: time.Now()}
	w.AddVote("sender-1", vote)
}
