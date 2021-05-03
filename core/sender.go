package core

import (
	"code.vegaprotocol.io/wendy/utils/list"
)

// Sender represents a node in the network and keeps track of the votes Wendy
// emits.
// Senders are not safe for concurrent access.
// NOTE: Since the sender never cleans up it's internal state, it always grow,
// hence, we might need to add a persisten storage.
type Sender struct {
	id    ID
	votes *list.List

	lastSeqSeen     uint64
	lastSeqCommited uint64

	commitedHashes map[Hash]struct{}
}

// NewSender returnsa new Sender instance.
func NewSender(id ID) *Sender {
	return &Sender{
		id:             id,
		votes:          list.New(),
		commitedHashes: make(map[Hash]struct{}),
	}
}

// LastSeqSeen returns the last higher consecutive Seq number registered by a vote.
func (s *Sender) LastSeqSeen() uint64 { return s.lastSeqSeen }

// LastSeqCommited returns XXX:
func (s *Sender) LastSeqCommited() uint64 { return s.lastSeqCommited }

// AddVote adds a vote to the vote list.
// Votes are insered in Seq ascendent order.
// It returns true if the vote hasn't been added before, otherwise, the vote is
// not added and false is returned.
func (s *Sender) AddVote(v *Vote) bool {
	item := s.votes.First(func(e *list.Element) bool {
		return e.Value.(*Vote).Seq >= v.Seq
	})

	// found a vote with equal or higher sequence number
	if item != nil {
		// duplicated seq number
		if item.Value.(*Vote).Seq == v.Seq {
			return false
		}
		item = s.votes.InsertBefore(v, item)
	} else {
		// no votes with higher Sequence number.
		item = s.votes.PushBack(v)
	}

	// update lastSeqSeen to the higher number before a gap is found.
	for e := item; e != nil; e = e.Next() {
		if v := e.Value.(*Vote).Seq; v == s.lastSeqSeen+1 {
			s.lastSeqSeen++
		}
	}

	return true
}

// AddVotes is a helper of AddVote to add more than one vote on a single call,
// it ignores the return value.
func (s *Sender) AddVotes(vs ...*Vote) {
	for _, v := range vs {
		s.AddVote(v)
	}
}

func elementByHash(hash Hash) list.FilterFunc {
	return func(e *list.Element) bool {
		return e.Value.(*Vote).TxHash == hash
	}
}

// Before returns true if tx1 has a lower sequence number than tx2.
// If tx1 and/or tx2 are not seen, Before returns false.
func (s *Sender) Before(tx1, tx2 Tx) bool {
	hash1, hash2 := tx1.Hash(), tx2.Hash()

	_, c1 := s.commitedHashes[hash1]
	_, c2 := s.commitedHashes[hash2]

	// both commited, does not make sense calling this when both are
	// commited.
	if c1 && c2 {
		return true
	}

	// tx1 commited, tx2 not commited
	if c1 && !c2 {
		return true
	}

	if !c1 && c2 {
		return false
	}

	v1 := s.votes.First(elementByHash(hash1))
	v2 := s.votes.First(elementByHash(hash2))

	// both voted
	if v1 != nil && v2 != nil {
		return v1.Value.(*Vote).Seq < v2.Value.(*Vote).Seq
	}

	// tx1 voted and tx2 not voted
	if v1 != nil && v2 == nil {
		return true
	}

	return false
}

// Seen returns whether a tx has been voted for or not.
// A Tx considered as seen iff there are no gaps befre the votes's seq number.
func (s *Sender) Seen(tx Tx) bool {
	hash := tx.Hash()
	item := s.votes.First(elementByHash(hash))
	if item == nil {
		return false
	}

	return item.Value.(*Vote).Seq <= s.lastSeqSeen
}

// UpdateTxSet will remove from its internal state all the references to a
// corresponging tx present in the txs argument.
// updateTxSet should be called when a new block is commited.
func (s *Sender) UpdateTxSet(txs ...Tx) {
	for _, tx := range txs {
		s.commitedHashes[tx.Hash()] = struct{}{}
	}
}
