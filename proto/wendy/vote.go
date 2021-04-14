package wendy

import (
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func NewVote(sender string, seq uint64, hash []byte) *Vote {
	vote := &Vote{
		Sender:   sender,
		Sequence: seq,
		TxHash:   hash,
		Seen:     timestamppb.Now(),
	}

	return vote
}
