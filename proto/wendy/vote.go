package wendy

import (
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func NewVote(seq uint64, hash []byte) *Vote {
	vote := &Vote{
		Sequence: seq,
		TxHash:   hash,
		Seen:     timestamppb.Now(),
	}

	return vote
}
