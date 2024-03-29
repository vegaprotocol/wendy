package wendy

import (
	"crypto/ed25519"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBlockedBy(t *testing.T) {
	w := New()
	w.UpdateValidatorSet([]Validator{
		pub0.Bytes(),
		pub1.Bytes(),
		pub2.Bytes(),
		pub3.Bytes(),
	})
	require.NotZero(t, w.HonestParties(), "can't run IsBlockedBy when HonestParties is zero")

	t.Run("1of4", func(t *testing.T) {
		var (
			vote0 = NewVote(pub0, 0, testTx0)
			vote1 = NewVote(pub0, 1, testTx1).WithPrevHash(vote0.Hash())
		)
		require.NoError(t, w.AddVotes(vote0, vote1))
		assert.True(t, w.IsBlockedBy(testTx0, testTx1), "should be blocked for HonestParties %d", w.HonestParties())
	})

	t.Run("2of4", func(t *testing.T) {
		var (
			vote0 = NewVote(pub1, 0, testTx0)
			vote1 = NewVote(pub1, 1, testTx1).WithPrevHash(vote0.Hash())
		)
		require.NoError(t, w.AddVotes(vote0, vote1))
		assert.True(t, w.IsBlockedBy(testTx0, testTx1), "should be blocked for HonestParties %d", w.HonestParties())
	})

	t.Run("3of4", func(t *testing.T) {
		var (
			vote0 = NewVote(pub2, 0, testTx0)
			vote1 = NewVote(pub2, 1, testTx1).WithPrevHash(vote0.Hash())
		)
		require.NoError(t, w.AddVotes(vote0, vote1))
		assert.False(t, w.IsBlockedBy(testTx0, testTx1), "should NOT be blocked for HonestParties %d", w.HonestParties())
	})

	t.Run("4of4", func(t *testing.T) {
		var (
			vote0 = NewVote(pub2, 0, testTx1) // these are in different order
			vote1 = NewVote(pub2, 1, testTx0).WithPrevHash(vote0.Hash())
		)
		require.NoError(t, w.AddVotes(vote0, vote1))
		assert.False(t, w.IsBlockedBy(testTx0, testTx1), "IsBlockedBy MUST be monotone")
	})
}

func TestVoteByHash(t *testing.T) {
	var (
		w    = New()
		tx   = testTx0
		vote = NewVote(NewPubkeyFromID("0xabcd"), 0, tx)
		hash = tx.Hash()
	)

	assert.Nil(t, w.VoteByTxHash(hash))

	require.NoError(t,
		w.AddVotes(vote),
	)
	got := w.VoteByTxHash(hash)

	assert.Equal(t, got, vote)
	assert.Nil(t, w.VoteByTxHash(testTx1.Hash()))
}

func TestIsBlocked(t *testing.T) {
	w := New()

	w.UpdateValidatorSet([]Validator{
		pub0.Bytes(), pub1.Bytes(), pub2.Bytes(), pub3.Bytes(),
	})
	require.NotZero(t, w.HonestParties(), "can't run IsBlockedBy when HonestParties is zero")

	require.NoError(t, w.AddVotes(NewVote(pub0, 0, testTx0)))
	require.True(t, w.IsBlocked(testTx0), "should be blocked with 1of4")

	require.NoError(t, w.AddVotes(NewVote(pub1, 0, testTx0)))
	require.True(t, w.IsBlocked(testTx0), "should be blocked with 2of4")

	require.NoError(t, w.AddVotes(NewVote(pub2, 0, testTx0)))
	require.False(t, w.IsBlocked(testTx0), "should be blocked with 3of4")

	t.Run("Gapped", func(t *testing.T) {
		tx := NewSimpleTx("tx-gapped", "hash-gapped")

		require.NoError(t, w.AddVotes(
			NewVote(pub0, 2, tx),
			NewVote(pub1, 2, tx),
			NewVote(pub2, 2, tx),
		))
		require.True(t, w.IsBlocked(tx), "should be blocked if seq is gapped")
	})
}

func newWendyFromTxsMap(t *testing.T, txsMap map[ID][]Tx) *Wendy {
	w := New()

	var nodes []Validator
	for node := range txsMap {
		nodes = append(nodes, Validator(node))
	}
	w.UpdateValidatorSet(nodes)

	// traverse the txsMaps in order to have determinism over tests.
	var keys sort.StringSlice = make([]string, 0, len(txsMap))
	for key := range txsMap {
		keys = append(keys, string(key))
	}
	sort.Sort(keys)

	for _, node := range keys {
		nodeId := ID(node)
		txs := txsMap[nodeId]
		// we use the tx index as seq

		var lastAddedVote *Vote
		for i, tx := range txs {
			seq := uint64(i)
			vote := NewVote(NewPubkeyFromID(nodeId), seq, tx)

			// since votes need to be linked by its hashes, we'll do it here.
			if lastAddedVote != nil {
				vote.WithPrevHash(lastAddedVote.Hash())
			}
			_, err := w.AddVote(vote)
			require.NoError(t, err)
			lastAddedVote = vote

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

		w := newWendyFromTxsMap(t,
			map[ID][]Tx{
				"0x00": {testTx1, testTx2, testTx3, testTx4, testTx5},
				"0x01": {testTx2, testTx3, testTx4, testTx5, testTx1},
				"0x02": {testTx3, testTx4, testTx5, testTx1, testTx2},
				"0x03": {testTx4, testTx5, testTx1, testTx2, testTx3},
				"0x04": {testTx5, testTx1, testTx2, testTx3, testTx4},
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

		w := newWendyFromTxsMap(t,
			map[ID][]Tx{
				"0x00": {testTx1, testTx2, testTx3, testTx4, testTx5},
				"0x01": {testTx1, testTx2, testTx3, testTx4, testTx5},
				"0x02": {testTx1, testTx2, testTx3, testTx4, testTx5},
				"0x03": {testTx1, testTx2, testTx3, testTx4, testTx5},
				"0x04": {testTx1, testTx2, testTx3, testTx4, testTx5},
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

func TestNewBlock(t *testing.T) {
	allTxs := []Tx{testTx0, testTx1, testTx2, testTx3, testTx4}
	w := newWendyFromTxsMap(t,
		map[ID][]Tx{
			"0x00": allTxs,
		},
	)

	block := w.NewBlock()
	assert.Equal(t, block.Txs, allTxs)

	t.Run("WithTxLimit", func(t *testing.T) {
		block := w.NewBlockWithOptions(
			NewBlockOptions{
				TxLimit: 3,
			},
		)

		assert.Len(t, block.Txs, 3)
		assert.Subset(t, allTxs, block.Txs, "Block Txs should be a subset of allTxs")
	})

	t.Run("WithMaxBlockSize", func(t *testing.T) {
		block := w.NewBlockWithOptions(
			NewBlockOptions{
				MaxBlockSize: 10,
			},
		)

		var size int
		for _, tx := range block.Txs {
			size += len(tx.Bytes())
		}
		assert.LessOrEqual(t, size, 10)
	})
}

func TestAddBlock(t *testing.T) {
	allTxs := []Tx{testTx0, testTx1, testTx2, testTx3, testTx4}
	w := newWendyFromTxsMap(t,
		map[ID][]Tx{
			"0x00": allTxs,
		},
	)

	// a new block is added with these txs, thus they should NOT be in the
	// NewBlock()
	block := &Block{
		Txs: []Tx{testTx0, testTx4},
	}
	w.AddBlock(block)

	newBlock := w.NewBlock()
	expectedTxs := []Tx{testTx1, testTx2, testTx3}
	require.Equal(t, expectedTxs, newBlock.Txs)
}

func TestVoteSigning(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(Rand)
	require.NoError(t, err)

	key := Pubkey(pub)
	vote := NewVote(key, 0, testTx0)
	sv := NewSignedVote(priv, vote)

	require.True(t, sv.Verify())

	sv.Data.Pubkey = pub0
	require.False(t, sv.Verify(), "verify should fails when pubkey updated")
}
