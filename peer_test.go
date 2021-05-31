package wendy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPeersVoting(t *testing.T) {
	t.Run("Adding", func(t *testing.T) {
		s := newTestPeer()
		tests := []struct {
			vote *Vote
			// expectations below
			lastSeq uint64
			added   bool
		}{
			// given a vote, lastSeq should be x and added or not.
			{vote: testVote0, lastSeq: 0, added: true},
			{vote: testVote1, lastSeq: 1, added: true},
			{vote: testVote2, lastSeq: 2, added: true},
			{vote: testVote4, lastSeq: 2, added: true},
			{vote: testVote3, lastSeq: 4, added: true},
			{vote: testVote3, lastSeq: 4, added: false},
		}

		for _, test := range tests {
			added, err := s.AddVote(test.vote)
			require.NoError(t, err)
			assert.Equal(t, test.lastSeq, s.LastSeqSeen(test.vote.Label))
			assert.Equal(t, test.added, added)
		}
	})

	t.Run("AddingWithWrongHashing", func(t *testing.T) {
		s := newTestPeer()
		require.NoError(t, s.AddVotes(testVote0))

		// next vote will have the wrong previous hash.
		newVote := *testVote1 // this creates a copy
		newVote.PrevHash = Checksum([]byte("wrong prev hash"))

		_, err := s.AddVote(&newVote)
		require.Error(t, err)
	})

	t.Run("AddingWithWrongHashingOnGap", func(t *testing.T) {
		var err error
		s := newTestPeer()

		wrongVote2 := *testVote2 // this creates a copy
		wrongVote2.PrevHash = Checksum([]byte("wrong prev hash"))

		err = s.AddVotes(testVote0, &wrongVote2)
		// testVote1 is missing so no error yet
		require.NoError(t, err)

		_, err = s.AddVote(testVote1)
		require.Error(t, err)
	})
}

func TestBefore(t *testing.T) {
	// List of priorities to evaluate t1 before t2
	//                               (t2)
	//                   +----------+-------+----------+
	//                   | Commited | Voted | NotSeen  |
	//       +-----------+----------+-------+----------+
	//       | Committed |    ?     | true  |   true   |
	// (t1)  | Voted     |  false   | seq() |   true   |
	//       | NotSeen   |  false   | false |   false  |
	//       +-----------+----------+-------+----------+
	//
	// Committed: The Tx has been committed by a Block.
	// Voted:     The Tx has been voted by a node.
	// NotSeen:   The Tx is not either commited or voted.
	//
	// seq(): sequence numbers need to be evaluated.
	// true:  `before()` returns always true.
	// false: `before()` returns always false.
	//
	// If t1 and t2 are both Commited, it shouldn't matter the answer since they are committed, the order is irrelevant.
	// If t1 is commited and t2 is Voted or NotSeen, t1 will be always before t2.
	// If t1 is voted, `before()` will return false if t1 is Commited or false if not seen.
	// If t1 is NotSeen, `before()` will return always false.
	// If t1 and t2 are both Voted, `before()` will return true if t1.Seq is < than t2.Seq.

	t.Run("OneCommited", func(t *testing.T) {
		s := newTestPeer()
		require.NoError(t,
			s.AddVotes(testVote0, testVote1, testVote2, testVote3, testVote4),
		)
		s.UpdateTxSet(testTx2)

		// tx1 commited
		assert.True(t, s.Before(testTx2, testTx0))
		assert.True(t, s.Before(testTx2, testTx1))
		assert.True(t, s.Before(testTx2, testTx3))
		assert.True(t, s.Before(testTx2, testTx4))

		// tx2 commited
		assert.False(t, s.Before(testTx0, testTx2))
		assert.False(t, s.Before(testTx1, testTx2))
		assert.False(t, s.Before(testTx3, testTx2))
		assert.False(t, s.Before(testTx4, testTx2))
	})

	t.Run("BothVoted", func(t *testing.T) {
		s := newTestPeer()
		require.NoError(t,
			s.AddVotes(testVote0, testVote1, testVote2, testVote3, testVote4),
		)

		assert.True(t, s.Before(testTx0, testTx1))
		assert.True(t, s.Before(testTx1, testTx2))
		assert.True(t, s.Before(testTx2, testTx3))
		assert.True(t, s.Before(testTx3, testTx4))
	})

	t.Run("NoneSeen", func(t *testing.T) {
		s := newTestPeer()
		assert.False(t, s.Before(testTx0, testTx1))
		assert.False(t, s.Before(testTx1, testTx0))
	})
}

func TestBeforeAcrossDifferentBucket(t *testing.T) {
	t.Run("Panic", func(t *testing.T) {
		s := newTestPeer()
		txA := NewSimpleTx("tx0", "h0").withLabel("A")
		txB := NewSimpleTx("tx1", "h1").withLabel("B")

		added, err := s.AddVote(NewVote(s.pub, 0, txA))
		require.NoError(t, err)
		require.True(t, added)

		added, err = s.AddVote(NewVote(s.pub, 0, txB))
		require.NoError(t, err)
		require.True(t, added)

		assert.Panics(t, func() {
			s.Before(txA, txB)
		})
	})
}
