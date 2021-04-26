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
