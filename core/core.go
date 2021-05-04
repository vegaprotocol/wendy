package core

import (
	"math"
	"sync"
)

type Wendy struct {
	validators []Validator
	quorum     int // quorum gets updated every time the validator set is updated.

	txsMtx sync.RWMutex
	txs    map[Hash]Tx

	votesMtx sync.RWMutex
	votes    map[Hash]*Vote
	peers    map[ID]*Peer
}

func New() *Wendy {
	return &Wendy{
		txs:   make(map[Hash]Tx),
		votes: make(map[Hash]*Vote),
		peers: make(map[ID]*Peer),
	}
}

// UpdateValidatorSet updates the list of validators in the consensus.
// Updating the validator set might affect the return value of Quorum().
// Upon updating the peers that are not in the new validator set are removed.
func (w *Wendy) UpdateValidatorSet(vs []Validator) {
	w.validators = vs

	q := math.Floor(
		float64(len(vs))*Quorum,
	) + 1
	w.quorum = int(q)

	w.votesMtx.Lock()
	defer w.votesMtx.Unlock()
	peers := make(map[ID]*Peer)
	// keep all the peers we already have and create new one if not present
	// those old peers that are not part of the new set will be discarded.
	for _, v := range vs {
		key := ID(v)
		if s, ok := w.peers[key]; ok {
			peers[key] = s
		} else {
			peers[key] = NewPeer(key)
		}
	}
	w.peers = peers
}

// HonestParties returns the required number of votes to be sure that at least
// one vote came from a honest validator.
// t + 1
func (w *Wendy) HonestParties() int {
	return w.quorum
}

// HonestMajority returns the minimum number of votes required to assure that I
// have a honest majority (2t + 1, which is equivalent to n-t). It's also the maximum number of honest parties I can
// expect to have.
func (w *Wendy) HonestMajority() int {
	return len(w.validators) - w.quorum
}

// AddTx adds a tx to the list of tx to be mined.
// AddTx returns false if the tx was already added.
// NOTE: This function is safe for concurrent access.
func (w *Wendy) AddTx(tx Tx) bool {
	w.txsMtx.Lock()
	defer w.txsMtx.Unlock()

	hash := tx.Hash()
	if _, ok := w.txs[hash]; ok {
		return false
	}
	w.txs[hash] = tx
	return true
}

// AddVote adds a vote to the list of votes.
// Votes are positioned given it's sequence number.
// AddVote returns alse if the vote was already added.
// NOTE: This function is safe for concurrent access.
func (w *Wendy) AddVote(v *Vote) bool {
	w.votesMtx.Lock()
	defer w.votesMtx.Unlock()

	// Register the vote on the peer
	peer, ok := w.peers[v.Pubkey]
	if !ok {
		peer = NewPeer(v.Pubkey)
		w.peers[v.Pubkey] = peer
	}

	// Register the vote based on its tx.Hash
	w.votes[v.TxHash] = v

	return peer.AddVote(v)
}

// CommitBlock iterate over the block's Txs set and remove them from Wendy's
// internal state.
// Txs present on block were probbaly added in the past via AddTx().
func (w *Wendy) CommitBlock(block Block) {
	w.votesMtx.Lock()
	defer w.votesMtx.Unlock()

	for _, peer := range w.peers {
		peer.UpdateTxSet(block.Txs...)
	}
}

// VoteByTxHash returns a vote given its tx.Hash
// Returns nil if the vote hasn't been seen.
// NOTE: This function is safe for concurrent access.
func (w *Wendy) VoteByTxHash(hash Hash) *Vote {
	w.votesMtx.RLock()
	defer w.votesMtx.RUnlock()
	return w.votes[hash]
}

// hasQuorum evaluates fn for every registered peer.
// It returns true if fn returned true at least w.Quorum() times.
// NOTE: This function is safe for concurrent access.
func (w *Wendy) hasQuorum(fn func(*Peer) bool) bool {
	w.votesMtx.RLock()
	defer w.votesMtx.RUnlock()

	var votes int
	for _, peer := range w.peers {
		if ok := fn(peer); ok {
			votes++
			if votes == w.quorum {
				return true
			}
		}
	}
	return false
}

// IsBlockedBy determines if tx2 might have priority over tx1.
// We say that tx1 is NOT blocked by tx2 if there are t+1 votes reporting tx1
// before tx2.
func (w *Wendy) IsBlockedBy(tx1, tx2 Tx) bool {
	// if there's no quorum that tx1 is before tx2, then tx1 is Blocked by tx2
	return !w.hasQuorum(func(p *Peer) bool {
		return p.Before(tx1, tx2)
	})
}

// IsBlocked identifies if it is pssible that a so-far-unknown transaction
// might be scheduled with priority to tx.
func (w *Wendy) IsBlocked(tx Tx) bool {
	// if there's no quorum that tx has been seen, then IsBlocked
	return !w.hasQuorum(func(p *Peer) bool {
		return p.Seen(tx)
	})
}

// Recompute is invoked on new{Block, Vote or Tx}
// returns
func (w *Wendy) Recompute() map[Hash][]Tx {
	// seen: tx1, tx2, tx3, tx4
	/* {
		"tx1": ["tx1", "tx2", "tx3", "tx4"],
		"tx2": ["tx2", "tx1", "tx3", "tx4"],
		"tx3": ["tx3", "tx1", "tx2", "tx4"],
		"tx4": ["tx4", "tx1", "tx2", "tx3"],
	}*/

	// 0. for every tx {
	// 1. Add myself to my blocking set
	// 2. I loop through my blocking set
	// 3. any one blocking someone from my blocking set is added to the blocking set
	// 4. goto 2. if I add a new tx to my blocking set.
	// }
	return nil
}
