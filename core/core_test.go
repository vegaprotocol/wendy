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
	label string
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
func (tx *testTx) Label() string { return tx.label }

func (tx *testTx) withLabel(l string) *testTx {
	tx.label = l
	return tx
}

func (tx *testTx) String() string {
	return fmt.Sprintf("%s (hash:%s)", string(tx.bytes), string(tx.hash))
}

// testTx<N> are used accross different tests
var (
	testTx0    = newTestTxStr("tx0", "h0")
	testTx1    = newTestTxStr("tx1", "h1")
	testTx2    = newTestTxStr("tx2", "h2")
	testTx3    = newTestTxStr("tx3", "h3")
	testTx4    = newTestTxStr("tx4", "h4")
	testTx5    = newTestTxStr("tx5", "h5")
	allTestTxs = []Tx{testTx0, testTx1, testTx2, testTx3, testTx4, testTx5}
)

func TestIsBlockedBy(t *testing.T) {
	w := New()
	w.UpdateValidatorSet([]Validator{
		"s0", "s1", "s2", "s3",
	})
	require.NotZero(t, w.HonestParties(), "can't run IsBlockedBy when HonestParties is zero")

	t.Run("1of4", func(t *testing.T) {
		w.AddVote(newVote("s0", 0, testTx0))
		w.AddVote(newVote("s0", 1, testTx1))
		assert.True(t, w.IsBlockedBy(testTx0, testTx1), "should be blocked for HonestParties %d", w.HonestParties())
	})

	t.Run("2of4", func(t *testing.T) {
		w.AddVote(newVote("s1", 0, testTx0))
		w.AddVote(newVote("s1", 1, testTx1))
		assert.True(t, w.IsBlockedBy(testTx0, testTx1), "should be blocked for HonestParties %d", w.HonestParties())
	})

	t.Run("3of4", func(t *testing.T) {
		w.AddVote(newVote("s2", 0, testTx0))
		w.AddVote(newVote("s2", 1, testTx1))
		assert.False(t, w.IsBlockedBy(testTx0, testTx1), "should NOT be blocked for HonestParties %d", w.HonestParties())
	})

	t.Run("4of4", func(t *testing.T) {
		w.AddVote(newVote("s2", 0, testTx1)) // these are in different order
		w.AddVote(newVote("s2", 1, testTx0))
		assert.False(t, w.IsBlockedBy(testTx0, testTx1), "IsBlockedBy MUST be monotone")
	})
}

func TestIsBlocked(t *testing.T) {
	w := New()

	w.UpdateValidatorSet([]Validator{
		"s0", "s1", "s2", "s3",
	})
	require.NotZero(t, w.HonestParties(), "can't run IsBlockedBy when HonestParties is zero")

	w.AddVote(newVote("s0", 0, testTx0))
	require.True(t, w.IsBlocked(testTx0), "should be blocked with 1of4")

	w.AddVote(newVote("s1", 0, testTx0))
	require.True(t, w.IsBlocked(testTx0), "should be blocked with 2of4")

	w.AddVote(newVote("s2", 0, testTx0))
	require.False(t, w.IsBlocked(testTx0), "should be blocked with 3of4")

	t.Run("Gapped", func(t *testing.T) {
		tx := newTestTxStr("tx-gapped", "hash-gapped")

		w.AddVote(newVote("s0", 2, tx))
		w.AddVote(newVote("s1", 2, tx))
		w.AddVote(newVote("s2", 2, tx))
		require.True(t, w.IsBlocked(tx), "should be blocked if seq is gapped")
	})
}

func newWendyFromTxsMap(txsMap map[ID][]Tx) *Wendy {
	w := New()

	var nodes []Validator
	for node := range txsMap {
		nodes = append(nodes, Validator(node))
	}
	w.UpdateValidatorSet(nodes)

	for node, txs := range txsMap {
		// we use the tx index as seq
		for i, tx := range txs {
			seq := uint64(i)
			w.AddVote(newVote(node, seq, tx))
			w.AddTx(tx)
		}
	}

	return w
}

func TestBlockingSet(t *testing.T) {
	t.Run("FairnessLoop", func(t *testing.T) {
		// For block order fairness, the most critical test case is a fairness
		// loop. Say we have four validators and four transactions, and the order
		// is:
		//         +-----------------------------+
		//         |       Sequence              |
		// +-------+-----+-----+-----+-----+-----+
		// | Node  |  0  |  1  |  2  |  3  |  4  |
		// +-------+-----+-----+-----+-----+-----+
		// | Node0 | Tx1 | Tx2 | Tx3 | Tx4 | Tx5 |
		// | Node1 | Tx2 | Tx3 | Tx4 | Tx5 | Tx1 |
		// | Node2 | Tx3 | Tx4 | Tx5 | Tx1 | Tx2 |
		// | Node3 | Tx4 | Tx5 | Tx1 | Tx2 | Tx3 |
		// | Node4 | Tx5 | Tx1 | Tx2 | Tx3 | Tx4 |
		// +-------+-----+-----+-----+-----+-----+
		//
		// In this case, we have a loop, that tx1 has priority over tx2, which has
		// priority over tx3, which has priority over tx4, which has priority over
		// tx1. This is, in fact, the reason we can only require that transactions
		// end up in the same block, rather than implementing an order right away.

		w := newWendyFromTxsMap(
			map[ID][]Tx{
				"Node0": {testTx1, testTx2, testTx3, testTx4, testTx5},
				"Node1": {testTx2, testTx3, testTx4, testTx5, testTx1},
				"Node2": {testTx3, testTx4, testTx5, testTx1, testTx2},
				"Node3": {testTx4, testTx5, testTx1, testTx2, testTx3},
				"Node4": {testTx5, testTx1, testTx2, testTx3, testTx4},
			},
		)

		// If tx2 IsBlockedBy tx1 we say that tx1 has priority over tx2.
		require.True(t, w.IsBlockedBy(testTx2, testTx1))
		require.True(t, w.IsBlockedBy(testTx3, testTx2))
		require.True(t, w.IsBlockedBy(testTx4, testTx3))
		require.True(t, w.IsBlockedBy(testTx1, testTx4))

		set := w.BlockingSet()

		// all txs depends on all txs, hence a loop exists
		allTxs := []Tx{testTx1, testTx2, testTx3, testTx4, testTx5}
		for _, tx := range allTxs {
			assert.ElementsMatch(t, set[tx.Hash()], allTxs)
		}
	})

	t.Run("FullyAgree", func(t *testing.T) {
		// For block order fairness, the most critical test case is a fairness
		// loop. Say we have four validators and four transactions, and the order
		// is:
		//         +-----------------------------+
		//         |       Sequence              |
		// +-------+-----+-----+-----+-----+-----+
		// | Node  |  0  |  1  |  2  |  3  |  4  |
		// +-------+-----+-----+-----+-----+-----+
		// | Node0 | Tx1 | Tx2 | Tx3 | Tx4 | Tx5 |
		// | Node1 | Tx1 | Tx2 | Tx3 | Tx4 | Tx5 |
		// | Node2 | Tx1 | Tx2 | Tx3 | Tx4 | Tx5 |
		// | Node3 | Tx1 | Tx2 | Tx3 | Tx4 | Tx5 |
		// | Node4 | Tx1 | Tx2 | Tx3 | Tx4 | Tx5 |
		// +-------+-----+-----+-----+-----+-----+
		//
		// In this case, we have a loop, that tx1 has priority over tx2, which has
		// priority over tx3, which has priority over tx4, which has priority over
		// tx1. This is, in fact, the reason we can only require that transactions
		// end up in the same block, rather than implementing an order right away.

		w := newWendyFromTxsMap(
			map[ID][]Tx{
				"Node0": {testTx1, testTx2, testTx3, testTx4, testTx5},
				"Node1": {testTx1, testTx2, testTx3, testTx4, testTx5},
				"Node2": {testTx1, testTx2, testTx3, testTx4, testTx5},
				"Node3": {testTx1, testTx2, testTx3, testTx4, testTx5},
				"Node4": {testTx1, testTx2, testTx3, testTx4, testTx5},
			},
		)

		set := w.BlockingSet()

		assert.ElementsMatch(t, set[testTx1.Hash()], []Tx{testTx1})
		assert.ElementsMatch(t, set[testTx2.Hash()], []Tx{testTx1, testTx2})
		assert.ElementsMatch(t, set[testTx3.Hash()], []Tx{testTx1, testTx2, testTx3})
		assert.ElementsMatch(t, set[testTx4.Hash()], []Tx{testTx1, testTx2, testTx3, testTx4})
		assert.ElementsMatch(t, set[testTx5.Hash()], []Tx{testTx1, testTx2, testTx3, testTx4, testTx5})
	})
}
