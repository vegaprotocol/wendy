package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var newTestPeer = func() *Peer { return NewPeer("xxx") }

var (
	tx0 = newTestTxStr("tx0", "h0")
	tx1 = newTestTxStr("tx1", "h1")
	tx2 = newTestTxStr("tx2", "h2")
	tx3 = newTestTxStr("tx3", "h3")
	tx4 = newTestTxStr("tx4", "h4")
)

func TestPeersVoting(t *testing.T) {
	t.Run("AddingVotes", func(t *testing.T) {
		s := newTestPeer()
		tests := []struct {
			vote *Vote
			// expectations below
			lastSeq uint64
			added   bool
		}{
			// given a vote, lastSeq should be x and added or not.
			{vote: newVote(s.id, 0, tx0), lastSeq: 0, added: true},
			{vote: newVote(s.id, 1, tx1), lastSeq: 1, added: true},
			{vote: newVote(s.id, 2, tx2), lastSeq: 2, added: true},
			{vote: newVote(s.id, 4, tx4), lastSeq: 2, added: true},
			{vote: newVote(s.id, 3, tx3), lastSeq: 4, added: true},
			{vote: newVote(s.id, 3, tx3), lastSeq: 4, added: false},
		}

		for _, test := range tests {
			added := s.AddVote(test.vote)
			assert.Equal(t, test.lastSeq, s.LastSeqSeen())
			assert.Equal(t, test.added, added)
		}
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
		s.AddVotes(
			newVote(s.id, 0, tx0),
			newVote(s.id, 1, tx1),
			newVote(s.id, 2, tx2),
			newVote(s.id, 3, tx3),
			newVote(s.id, 4, tx4),
		)
		s.UpdateTxSet(tx2)

		// tx1 commited
		assert.True(t, s.Before(tx2, tx0))
		assert.True(t, s.Before(tx2, tx1))
		assert.True(t, s.Before(tx2, tx3))
		assert.True(t, s.Before(tx2, tx4))

		// tx2 commited
		assert.False(t, s.Before(tx0, tx2))
		assert.False(t, s.Before(tx1, tx2))
		assert.False(t, s.Before(tx3, tx2))
		assert.False(t, s.Before(tx4, tx2))
	})

	t.Run("BothVoted", func(t *testing.T) {
		s := newTestPeer()
		s.AddVotes(
			newVote(s.id, 0, tx0),
			newVote(s.id, 1, tx1),
			newVote(s.id, 2, tx2),
			newVote(s.id, 3, tx3),
			newVote(s.id, 4, tx4),
		)

		assert.True(t, s.Before(tx0, tx1))
		assert.True(t, s.Before(tx1, tx2))
		assert.True(t, s.Before(tx2, tx3))
		assert.True(t, s.Before(tx3, tx4))
	})

	t.Run("NoneSeen", func(t *testing.T) {
		s := newTestPeer()
		assert.False(t, s.Before(tx0, tx1))
		assert.False(t, s.Before(tx1, tx0))
	})
}
