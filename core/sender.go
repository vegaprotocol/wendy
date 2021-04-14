package core

import (
	"container/list"
)

type Sender struct {
	id      ID
	votes   *list.List
	pending *list.List
}

func NewSender(id ID) *Sender {
	return &Sender{
		id:      id,
		votes:   list.New(),
		pending: list.New(),
	}
}

// NextSeq returns the next expected sequence number.
// Seq numbers starts with 0.
func (s *Sender) NextSeq() uint64 {
	el := s.votes.Back()
	if el == nil {
		return 0
	}
	return el.Value.(*Vote).Seq + 1
}

// AddVote adds a vote to the vote list.
// If the vote has been already inserted, it's ignored.
// If the vote's Seq is gapped it's added to the pending list.
// Everytime a vote is added the pending list if checked to see if we can add
// them too.
func (s *Sender) AddVote(v *Vote) {
	nextSeq := s.NextSeq()

	// vote already inserted, ignore.
	if v.Seq < nextSeq {
		return
	}

	// there is a gap between nextSeq and the vote, so we need to add it to the
	// pending list but making sure it's not duplicated.
	if v.Seq > nextSeq {
		for e := s.pending.Front(); e != nil; e = e.Next() {
			seq := e.Value.(*Vote).Seq
			// already inserted, ignore.
			if seq == v.Seq {
				return
			}

			// insert right before the next vote with higher seq number.
			if seq > v.Seq {
				s.pending.InsertBefore(v, e)
				return
			}
		}
		// list was empty
		s.pending.PushBack(v)
		return
	}

	// vote has the expected seq number.
	s.votes.PushBack(v)

	// check the pending list and add the possible votes that are correlated.
	var next *list.Element
	for e := s.pending.Front(); e != nil; e = next {
		next = e.Next()

		next := e.Value.(*Vote)
		nextSeq = s.NextSeq()
		if nextSeq == next.Seq {
			s.votes.PushBack(next)
			s.pending.Remove(e)
		}

		// gap found
		if nextSeq < next.Seq {
			break
		}
	}
}
