package wendy

import (
	"errors"

	"github.com/vegaprotocol/wendy/utils/list"
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
	pub     Pubkey
	buckets map[string]*peerBucket
}

// NewPeer returnsa new Peer instance.
func NewPeer(pub Pubkey) *Peer {
	return &Peer{
		pub:     pub,
		buckets: make(map[string]*peerBucket),
	}
}

// bucket returns the peerBucket corresponding to a given label.
// A new peerBucket is created if it does not exist.
func (p *Peer) bucket(label string) *peerBucket {
	b, ok := p.buckets[label]
	if !ok {
		b = newPeerBucket()
		p.buckets[label] = b
	}
	return b
}

// LastSeqSeen returns the last higher consecutive Seq number registered by a vote.
func (p *Peer) LastSeqSeen(label string) uint64 { return p.bucket(label).lastSeqSeen }

// AddVote adds a vote to the vote list.
// It returns true if the vote hasn't been added before, otherwise, the vote is
// not added and false is returned.
func (p *Peer) AddVote(v *Vote) (bool, error) {
	bucket := p.bucket(v.Label)

	// Since is most likely that votes are inserted in order (lower to higher
	// seq numbers), we lookup the previous vote traversing the
	// list list backwards (tail to head) so that if we insert vote with seq =
	// N, we are looking for a vote with seq N or lower.
	item := bucket.votes.First(func(e *list.Element) bool {
		return e.Value.(*Vote).Seq <= v.Seq
	}, list.Backward)

	// found a vote with lower or equal sequence number
	if item != nil {
		prev := item.Value.(*Vote)

		// duplicated seq number
		if prev.Seq == v.Seq {
			return false, nil
		}

		// Validate hash linking
		// We need to perform 2 validations:
		// 1. added vote against its previous one: (prev.Hash() == addedVote.PrevHash)
		if err := validHashes(prev, v); err != nil {
			return false, err
		}

		item = bucket.votes.InsertAfter(v, item)

		// 2. added vote against its next one:     (addedVote.Hash() == next.PrevHash)
		if next := item.Next(); next != nil {
			if err := validHashes(v, next.Value.(*Vote)); err != nil {
				return false, err
			}
		}

	} else {
		// no votes with Sequence number.
		// send it to the beginning of the list.
		item = bucket.votes.PushBack(v)
	}

	// update lastSeqSeen to the higher number before a gap is found.
	for e := item; e != nil; e = e.Next() {
		if v := e.Value.(*Vote).Seq; v == bucket.lastSeqSeen+1 {
			bucket.lastSeqSeen++
		}
	}

	return true, nil
}

func validHashes(prev, next *Vote) error {
	// Only validate hashes when votes's Seq numbers are contiguous.
	if prev.Seq+1 != next.Seq {
		return nil
	}

	if prev.Hash() != next.PrevHash {
		return errors.New("hashes don't match")
	}

	return nil
}

// AddVotes is a helper of AddVote to add more than one vote on a single call,
// it ignores the return value.
func (p *Peer) AddVotes(vs ...*Vote) error {
	for _, v := range vs {
		if _, err := p.AddVote(v); err != nil {
			return err
		}
	}
	return nil
}

func elementByHash(hash Hash) list.FilterFunc {
	return func(e *list.Element) bool {
		return e.Value.(*Vote).TxHash == hash
	}
}

// Before returns true if tx1 has a lower sequence number than tx2.
// If tx1 and/or tx2 are not seen, Before returns false.
// Txs MUST belong to the same Label() otherwise Before will panic.
func (p *Peer) Before(tx1, tx2 Tx) bool {
	if tx1.Label() != tx2.Label() {
		panic("labels can't be different")
	}

	bucket := p.bucket(tx1.Label())
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
func (p *Peer) Seen(tx Tx) bool {
	bucket := p.bucket(tx.Label())
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
func (p *Peer) UpdateTxSet(txs ...Tx) {
	for _, tx := range txs {
		p.bucket(tx.Label()).commitedHashes[tx.Hash()] = struct{}{}
	}
}
