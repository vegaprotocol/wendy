package core

import (
	"code.vegaprotocol.io/wendy/utils/list"
)

// Peer represents a node in the network and keeps track of the votes Wendy
// emits.
// Peers are not safe for concurrent access.
// NOTE: Since the Peer never cleans up it's internal state, it always grow,
// hence, we might need to add a persistent storage.
type Peer struct {
	id    ID
	votes *list.List

	lastSeqSeen     uint64
	lastSeqCommited uint64

	// D (delivered on the paper)
	commitedHashes map[Hash]struct{}
}

// NewPeer returnsa new Peer instance.
func NewPeer(id ID) *Peer {
	return &Peer{
		id:             id,
		votes:          list.New(),
		commitedHashes: make(map[Hash]struct{}),
	}
}

// LastSeqSeen returns the last higher consecutive Seq number registered by a vote.
func (r *Peer) LastSeqSeen() uint64 { return r.lastSeqSeen }

// LastSeqCommited returns XXX:
func (r *Peer) LastSeqCommited() uint64 { return r.lastSeqCommited }

// AddVote adds a vote to the vote list.
// Votes are inserted in descendent order by Seq (higher sequence on the fron,
// lower on the back).
// It returns true if the vote hasn't been added before, otherwise, the vote is
// not added and false is returned.
func (r *Peer) AddVote(v *Vote) bool {
	item := r.votes.First(func(e *list.Element) bool {
		return e.Value.(*Vote).Seq <= v.Seq
	})

	// found a vote with equal or higher sequence number
	if item != nil {
		// duplicated seq number
		if item.Value.(*Vote).Seq == v.Seq {
			return false
		}
		item = r.votes.InsertBefore(v, item)
	} else {
		// no votes with higher Sequence number.
		item = r.votes.PushFront(v)
	}

	// update lastSeqSeen to the higher number before a gap is found.
	for e := item; e != nil; e = e.Prev() {
		if v := e.Value.(*Vote).Seq; v == r.lastSeqSeen+1 {
			r.lastSeqSeen++
		}
	}

	return true
}

// AddVotes is a helper of AddVote to add more than one vote on a single call,
// it ignores the return value.
func (r *Peer) AddVotes(vs ...*Vote) {
	for _, v := range vs {
		r.AddVote(v)
	}
}

func elementByHash(hash Hash) list.FilterFunc {
	return func(e *list.Element) bool {
		return e.Value.(*Vote).TxHash == hash
	}
}

// Before returns true if tx1 has a lower sequence number than tx2.
// If tx1 and/or tx2 are not seen, Before returns false.
func (r *Peer) Before(tx1, tx2 Tx) bool {
	hash1, hash2 := tx1.Hash(), tx2.Hash()

	_, c1 := r.commitedHashes[hash1]
	_, c2 := r.commitedHashes[hash2]

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

	v1 := r.votes.First(elementByHash(hash1))
	v2 := r.votes.First(elementByHash(hash2))

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
func (r *Peer) Seen(tx Tx) bool {
	hash := tx.Hash()
	item := r.votes.First(elementByHash(hash))
	if item == nil {
		return false
	}

	return item.Value.(*Vote).Seq <= r.lastSeqSeen
}

// UpdateTxSet will remove from its internal state all the references to a
// corresponging tx present in the txs argument.
// updateTxSet should be called when a new block is commited.
func (r *Peer) UpdateTxSet(txs ...Tx) {
	for _, tx := range txs {
		r.commitedHashes[tx.Hash()] = struct{}{}
	}
}
