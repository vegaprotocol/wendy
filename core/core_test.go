package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func newTestTxStr(bytes, hash string) *testTx {
	return &testTx{bytes: []byte(bytes), hash: []byte(hash)}
}

func TestSequenceConsensus(t *testing.T) {
	w := New()

	// these are the validators from the network
	w.UpdateValidatorSet([]Validator{
		"vA", "vB", "vC",
	})

	/* votes
	a: [tx0, tx1, tx3, tx2] -> voterA is cheating (tx3 before tx2)
	b: [tx0, tx1, tx2, tx3]
	c: [tx0, tx1, tx2, tx3]
	*/

	tx0 := newTestTxStr("tx0", "hash0")
	tx1 := newTestTxStr("tx1", "hash1")
	tx2 := newTestTxStr("tx2", "hash2")
	tx3 := newTestTxStr("tx3", "hash3")

	// Tx0
	w.AddVote("vA", newVote(0, tx0))
	w.AddVote("vB", newVote(0, tx0))
	w.AddVote("vC", newVote(0, tx0))

	// Tx1
	w.AddVote("vA", newVote(1, tx1))
	w.AddVote("vB", newVote(1, tx1))
	w.AddVote("vC", newVote(1, tx1))

	// Tx2,3
	w.AddVote("vA", newVote(2, tx3)) // cheating tx3
	w.AddVote("vB", newVote(2, tx2))
	w.AddVote("vC", newVote(2, tx2))

	// Tx3,2
	w.AddVote("vA", newVote(3, tx2)) // cheating tx2
	w.AddVote("vB", newVote(3, tx3))
	w.AddVote("vC", newVote(3, tx3))
}

func TestIsBlockedBy(t *testing.T) {
	w := New()
	w.UpdateValidatorSet([]Validator{
		"s0", "s1", "s2", "s3",
	})
	require.NotZero(t, w.Quorum(), "can't run IsBlockedBy when Quorum is zero")

	tx0, tx1 :=
		newTestTxStr("tx0", "h0"),
		newTestTxStr("tx1", "h1")

	t.Run("1of4", func(t *testing.T) {
		w.AddVote("s0", newVote(0, tx0))
		w.AddVote("s0", newVote(1, tx1))
		assert.True(t, w.IsBlockedBy(tx0, tx1), "should be blocked for Quorum %d", w.Quorum())
	})

	t.Run("2of4", func(t *testing.T) {
		w.AddVote("s1", newVote(0, tx0))
		w.AddVote("s1", newVote(1, tx1))
		assert.True(t, w.IsBlockedBy(tx0, tx1), "should be blocked for Quorum %d", w.Quorum())
	})

	t.Run("3of4", func(t *testing.T) {
		w.AddVote("s2", newVote(0, tx0))
		w.AddVote("s2", newVote(1, tx1))
		assert.False(t, w.IsBlockedBy(tx0, tx1), "should NOT be blocked for Quorum %d", w.Quorum())
	})

	t.Run("4of4", func(t *testing.T) {
		w.AddVote("s2", newVote(0, tx1)) // these are in different order
		w.AddVote("s2", newVote(1, tx0))
		assert.False(t, w.IsBlockedBy(tx0, tx1), "IsBlockedBy MUST be monotone")
	})
}

func TestIsBlocked(t *testing.T) {
	w := New()

	w.UpdateValidatorSet([]Validator{
		"s0", "s1", "s2", "s3",
	})
	require.NotZero(t, w.Quorum(), "can't run IsBlockedBy when Quorum is zero")

	tx0 := newTestTxStr("tx0", "h0")

	w.AddVote("s0", newVote(0, tx0))
	require.True(t, w.IsBlocked(tx0), "should be blocked with 1of4")

	w.AddVote("s1", newVote(0, tx0))
	require.True(t, w.IsBlocked(tx0), "should be blocked with 2of4")

	w.AddVote("s2", newVote(0, tx0))
	require.False(t, w.IsBlocked(tx0), "should be blocked with 3of4")

}
