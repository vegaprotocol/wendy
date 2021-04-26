package core

import (
	"container/list"
	"math"
)

type Sender struct {
	id      ID
	votes   *list.List
	pending *list.List

	// NOTE: Use a LRU?
	seen map[Hash]struct{}
}

func NewSender(id ID) *Sender {
	return &Sender{
		id:      id,
		votes:   list.New(),
		pending: list.New(),
		seen:    make(map[Hash]struct{}),
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
func (s *Sender) AddVote(v *Vote) bool {
	nextSeq := s.NextSeq()

	// vote already inserted, ignore.
	if v.Seq < nextSeq {
		return false
	}

	// there is a gap between nextSeq and the vote, so we need to add it to the
	// pending list but making sure it's not duplicated.
	if v.Seq > nextSeq {
		for e := s.pending.Front(); e != nil; e = e.Next() {
			seq := e.Value.(*Vote).Seq
			// already inserted, ignore.
			if seq == v.Seq {
				return false
			}

			// insert right before the next vote with higher seq number.
			if seq > v.Seq {
				s.pending.InsertBefore(v, e)
				return true
			}
		}
		// list was empty
		s.pending.PushBack(v)
		return true
	}

	// vote has the expected seq number.
	s.votes.PushBack(v)

	// we consider a vote `seen` only when its seq number is last+1.
	s.seen[v.TxHash] = struct{}{}

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

	return true
}

// Before returns true if tx1 has a lower sequence number than tx2.
// If tx1 and/or tx2 are not seen, Before returns false.
func (s *Sender) Before(tx1, tx2 Tx) bool {
	var (
		// we assume false (see `seq1 < seq2` below) until proven otherwise.
		seq1 uint64 = math.MaxUint64
		seq2 uint64
	)
	h1, h2 := tx1.Hash(), tx2.Hash()

	for e := s.votes.Front(); e != nil; e = e.Next() {
		v := e.Value.(*Vote)
		if v.TxHash == h1 {
			seq1 = v.Seq
		} else if v.TxHash == h2 {
			seq2 = v.Seq
		}

		if seq1 < seq2 {
			return true
		}
	}
	return false
}

func (s *Sender) Seen(tx Tx) bool {
	_, ok := s.seen[tx.Hash()]
	return ok
}
