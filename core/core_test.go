package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ Tx = &testTx{}

type testTx struct {
	bytes []byte
	hash  []byte
}

func newTestTxStr(bytes, hash string) *testTx {
	return &testTx{bytes: []byte(bytes), hash: []byte(hash)}
}

func (tx *testTx) Bytes() []byte { return tx.bytes }
func (tx *testTx) Hash() Hash {
	var hash Hash
	copy(hash[:], tx.hash)
	return hash
}

func (tx *testTx) String() string {
	return fmt.Sprintf("%s (hash:%s)", string(tx.bytes), string(tx.hash))
}

// txN are used accross different tests
var (
	tx0 = newTestTxStr("tx0", "h0")
	tx1 = newTestTxStr("tx1", "h1")
	tx2 = newTestTxStr("tx2", "h2")
	tx3 = newTestTxStr("tx3", "h3")
	tx4 = newTestTxStr("tx4", "h4")
)

func TestIsBlockedBy(t *testing.T) {
	w := New()
	w.UpdateValidatorSet([]Validator{
		"s0", "s1", "s2", "s3",
	})
	require.NotZero(t, w.HonestParties(), "can't run IsBlockedBy when HonestParties is zero")

	tx0, tx1 :=
		newTestTxStr("tx0", "h0"),
		newTestTxStr("tx1", "h1")

	t.Run("1of4", func(t *testing.T) {
		w.AddVote(newVote("s0", 0, tx0))
		w.AddVote(newVote("s0", 1, tx1))
		assert.True(t, w.IsBlockedBy(tx0, tx1), "should be blocked for HonestParties %d", w.HonestParties())
	})

	t.Run("2of4", func(t *testing.T) {
		w.AddVote(newVote("s1", 0, tx0))
		w.AddVote(newVote("s1", 1, tx1))
		assert.True(t, w.IsBlockedBy(tx0, tx1), "should be blocked for HonestParties %d", w.HonestParties())
	})

	t.Run("3of4", func(t *testing.T) {
		w.AddVote(newVote("s2", 0, tx0))
		w.AddVote(newVote("s2", 1, tx1))
		assert.False(t, w.IsBlockedBy(tx0, tx1), "should NOT be blocked for HonestParties %d", w.HonestParties())
	})

	t.Run("4of4", func(t *testing.T) {
		w.AddVote(newVote("s2", 0, tx1)) // these are in different order
		w.AddVote(newVote("s2", 1, tx0))
		assert.False(t, w.IsBlockedBy(tx0, tx1), "IsBlockedBy MUST be monotone")
	})
}

func TestIsBlocked(t *testing.T) {
	w := New()

	w.UpdateValidatorSet([]Validator{
		"s0", "s1", "s2", "s3",
	})
	require.NotZero(t, w.HonestParties(), "can't run IsBlockedBy when HonestParties is zero")

	tx0 := newTestTxStr("tx0", "h0")

	w.AddVote(newVote("s0", 0, tx0))
	require.True(t, w.IsBlocked(tx0), "should be blocked with 1of4")

	w.AddVote(newVote("s1", 0, tx0))
	require.True(t, w.IsBlocked(tx0), "should be blocked with 2of4")

	w.AddVote(newVote("s2", 0, tx0))
	require.False(t, w.IsBlocked(tx0), "should be blocked with 3of4")

	t.Run("Gapped", func(t *testing.T) {
		tx := newTestTxStr("tx-gapped", "hash-gapped")

		w.AddVote(newVote("s0", 2, tx))
		w.AddVote(newVote("s1", 2, tx))
		w.AddVote(newVote("s2", 2, tx))
		require.True(t, w.IsBlocked(tx), "should be blocked if seq is gapped")
	})
}

func TestFairnessLoop(t *testing.T) {
	// For block order fairness, the most critical test case is a fairness
	// loop. Say we have four validators and four transactions, and the order
	// is:
	//         +-----------------------+
	//         |       Sequence        |
	// +-------+-----+-----+-----+-----+
	// | Node  |  0  |  1  |  2  |  3  |
	// +-------+-----+-----+-----+-----+
	// | Node0 | Tx1 | Tx2 | Tx3 | Tx4 |
	// | Node1 | Tx2 | Tx3 | Tx4 | Tx1 |
	// | Node2 | Tx3 | Tx4 | Tx1 | Tx2 |
	// | Node3 | Tx4 | Tx1 | Tx2 | Tx3 |
	// +-------+-----+-----+-----+-----+
	//
	// In this case, we have a loop, that tx1 has priority over tx2, which has
	// priority over tx3, which has priority over tx4, which has priority over
	// tx1. This is, in fact, the reason we can only require that transactions
	// end up in the same block, rather than implementing an order right away.

	w := New()
	w.UpdateValidatorSet([]Validator{
		"Node0", "Node1", "Node2", "Node3",
	})

	txsMap := map[ID][]Tx{
		"Node0": {tx1, tx2, tx3, tx4},
		"Node1": {tx2, tx3, tx4, tx1},
		"Node2": {tx3, tx4, tx1, tx2},
		"Node3": {tx4, tx1, tx2, tx3},
	}

	for node, txs := range txsMap {
		// we use the tx index as seq
		for i, tx := range txs {
			seq := uint64(i)
			t.Logf("%s: AddVote(seq=%d, tx=%s)", node, seq, tx)
			w.AddVote(newVote(node, seq, tx))
		}
		t.Logf("")
	}

	// If tx2 IsBlockedBy tx1 we say that tx1 has priority over tx2.
	require.True(t, w.IsBlockedBy(tx2, tx1))
	require.True(t, w.IsBlockedBy(tx3, tx2))
	require.True(t, w.IsBlockedBy(tx4, tx3))
	require.True(t, w.IsBlockedBy(tx1, tx4))
}
