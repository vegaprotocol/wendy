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
