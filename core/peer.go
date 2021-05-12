package core

import (
	"code.vegaprotocol.io/wendy/utils/list"
)

// peerBucket holds the state of a peer for a given label.
// Peers have one bucket per label.
type peerBucket struct {
	votes          *list.List
	lastSeqSeen    uint64
	commitedHashes map[Hash]struct{}
}

// newPeerBucket returns an initialized peerBucket.
func newPeerBucket() *peerBucket {
	return &peerBucket{
		votes:          list.New(),
		commitedHashes: make(map[Hash]struct{}),
	}
}

// Peer represents a node in the network and keeps track of the votes Wendy
// emits.
// The Peer stores one state per label in a peerBucket.
// Peers are not safe for concurrent access.
// NOTE: Since the Peer never cleans up it's internal state, it always grow,
// hence, we might need to add a persistent storage.
type Peer struct {
	id      ID
	buckets map[string]*peerBucket
}

// NewPeer returnsa new Peer instance.
func NewPeer(id ID) *Peer {
	return &Peer{
		id:      id,
		buckets: make(map[string]*peerBucket),
	}
}

// bucket returns the peerBucket corresponding to a given label.
// A new peerBucket is created if it does not exist.
func (r *Peer) bucket(label string) *peerBucket {
	b, ok := r.buckets[label]
	if !ok {
		b = newPeerBucket()
		r.buckets[label] = b
	}
	return b
}

// LastSeqSeen returns the last higher consecutive Seq number registered by a vote.
func (r *Peer) LastSeqSeen(label string) uint64 { return r.bucket(label).lastSeqSeen }

// AddVote adds a vote to the vote list.
// Votes are inserted in descendent order by Seq (higher sequence on the fron,
// lower on the back).
// It returns true if the vote hasn't been added before, otherwise, the vote is
// not added and false is returned.
func (r *Peer) AddVote(v *Vote) bool {
	bucket := r.bucket(v.Label)

	item := bucket.votes.First(func(e *list.Element) bool {
		return e.Value.(*Vote).Seq <= v.Seq
	})

	// found a vote with equal or higher sequence number
	if item != nil {
		// duplicated seq number
		if item.Value.(*Vote).Seq == v.Seq {
			return false
		}
		item = bucket.votes.InsertBefore(v, item)
	} else {
		// no votes with higher Sequence number.
		item = bucket.votes.PushFront(v)
	}

	// update lastSeqSeen to the higher number before a gap is found.
	for e := item; e != nil; e = e.Prev() {
		if v := e.Value.(*Vote).Seq; v == bucket.lastSeqSeen+1 {
			bucket.lastSeqSeen++
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
// Txs MUST belong to the same Label() otherwise Before will panic.
func (r *Peer) Before(tx1, tx2 Tx) bool {
	if tx1.Label() != tx2.Label() {
		panic("labels can't be different")
	}

	bucket := r.bucket(tx1.Label())
	hash1, hash2 := tx1.Hash(), tx2.Hash()

	_, c1 := bucket.commitedHashes[hash1]
	_, c2 := bucket.commitedHashes[hash2]

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

	v1 := bucket.votes.First(elementByHash(hash1))
	v2 := bucket.votes.First(elementByHash(hash2))

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
	bucket := r.bucket(tx.Label())
	hash := tx.Hash()
	item := bucket.votes.First(elementByHash(hash))
	if item == nil {
		return false
	}

	return item.Value.(*Vote).Seq <= bucket.lastSeqSeen
}

// UpdateTxSet will remove from its internal state all the references to a
// corresponging tx present in the txs argument.
// updateTxSet should be called when a new block is commited.
func (r *Peer) UpdateTxSet(txs ...Tx) {
	for _, tx := range txs {
		r.bucket(tx.Label()).commitedHashes[tx.Hash()] = struct{}{}
	}
}
