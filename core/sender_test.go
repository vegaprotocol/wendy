package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendersVoting(t *testing.T) {
	s := NewSender("xxx")
	assert.Zero(t, s.NextSeq())

	s.AddVote(&Vote{Seq: 0})
	assert.EqualValues(t, 1, s.NextSeq())

	s.AddVote(&Vote{Seq: 0})
	assert.EqualValues(t, 1, s.NextSeq(), "AddVote should be idempotent")

	s.AddVote(&Vote{Seq: 1})
	assert.EqualValues(t, 2, s.NextSeq(), "Vote with seq+1 should increment NextSeq")

	// gaps
	s.AddVote(&Vote{Seq: 3})
	assert.EqualValues(t, 2, s.NextSeq(), "Gapped vote should not incrment NextSeq")
	s.AddVote(&Vote{Seq: 4})
	assert.EqualValues(t, 2, s.NextSeq(), "Gapped vote should not increment NextSeq")
	s.AddVote(&Vote{Seq: 6})

	s.AddVote(&Vote{Seq: 2})
	assert.EqualValues(t, 5, s.NextSeq(), "Vote with seq+1 should consume pending votes")

	s.AddVote(&Vote{Seq: 5})
	assert.EqualValues(t, 7, s.NextSeq(), "Gappend vote should consume pending vote")
}

func TestSendersVotingCache(t *testing.T) {
	s := NewSender("xxx")

	assert.True(t, s.AddVote(&Vote{Seq: 0}))
	assert.False(t, s.AddVote(&Vote{Seq: 0}))

	assert.True(t, s.AddVote(&Vote{Seq: 1}))
	assert.False(t, s.AddVote(&Vote{Seq: 1}))

	assert.True(t, s.AddVote(&Vote{Seq: 3}))
	assert.False(t, s.AddVote(&Vote{Seq: 3}))

	assert.True(t, s.AddVote(&Vote{Seq: 5}))
	assert.True(t, s.AddVote(&Vote{Seq: 4}))
}

func TestSenderBeforeTx1Tx2(t *testing.T) {
	var (
		tx0 = newTestTxStr("tx0", "h0")
		tx1 = newTestTxStr("tx1", "h1")
		tx2 = newTestTxStr("tx2", "h2")
		tx3 = newTestTxStr("tx3", "h3")
	)

	t.Run("With no votes", func(t *testing.T) {
		s := NewSender("xxx")
		assert.False(t, s.Before(tx0, tx1))
		assert.False(t, s.Before(tx1, tx0))
		assert.False(t, s.Before(tx2, tx3))
		assert.False(t, s.Before(tx3, tx2))
	})

	t.Run("WithPartialVotes", func(t *testing.T) {
		s := NewSender("xxx")

		s.AddVote(newVote("xxx", 0, tx0))
		assert.False(t, s.Before(tx0, tx1))
		assert.False(t, s.Before(tx1, tx0))

		s.AddVote(newVote("xxx", 0, tx3))
		assert.False(t, s.Before(tx2, tx3))
		assert.False(t, s.Before(tx3, tx2))
	})

	t.Run("FullVotes", func(t *testing.T) {
		s := NewSender("xxx")

		s.AddVote(newVote("xxx", 0, tx0))
		s.AddVote(newVote("xxx", 1, tx1))
		s.AddVote(newVote("xxx", 2, tx2))
		s.AddVote(newVote("xxx", 3, tx3))

		assert.True(t, s.Before(tx0, tx1))
		assert.True(t, s.Before(tx0, tx2))
		assert.True(t, s.Before(tx1, tx2))
		assert.True(t, s.Before(tx1, tx3))

		assert.False(t, s.Before(tx1, tx0))
	})

}
