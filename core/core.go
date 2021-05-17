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
	for _, val := range vs {
		key := Pubkey(val)
		id := ID(key.String())
		if s, ok := w.peers[id]; ok {
			peers[id] = s
		} else {
			peers[id] = NewPeer(key)
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

	key := ID(v.Pubkey.String())
	// Register the vote on the peer
	peer, ok := w.peers[key]
	if !ok {
		pub := NewPubkeyFromID(key)
		peer = NewPeer(pub)
		w.peers[key] = peer
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

// BlockingSet should be invoked on every new{Block, Vote or Tx} and returns a
// list of blocking Txs for all the currently seen Txs.
func (w *Wendy) BlockingSet() BlockingSet {
	// Copy all tx from the local map to an array
	var txs []Tx = make([]Tx, 0, len(w.txs))
	w.txsMtx.RLock()
	for _, tx := range w.txs {
		txs = append(txs, tx)
	}
	w.txsMtx.RUnlock()

	// Build the dependency matrix for all Txs
	var matrix [][]bool = make([][]bool, len(txs))
	for i := range matrix {
		matrix[i] = make([]bool, len(txs))
	}
	for i, tx1 := range txs {
		for j, tx2 := range txs {
			matrix[i][j] = w.IsBlockedBy(tx1, tx2)
		}
	}

	set := BlockingSet{}
	for i, tx := range txs {
		blockers := []Tx{}
		deps := make(map[int]struct{})
		recompute(matrix, i, deps)

		for txIndex := range deps {
			blockers = append(blockers, txs[txIndex])
		}

		set[tx.Hash()] = blockers
	}
	return set
}

func recompute(matrix [][]bool, index int, deps map[int]struct{}) {
	row := matrix[index]

	for i := 0; i < len(row); i++ {
		if _, ok := deps[i]; ok {
			continue
		}

		// matrix diagonal
		if i == index {
			deps[i] = struct{}{}
			continue
		}

		if row[i] {
			deps[i] = struct{}{}
			recompute(matrix, i, deps)
		}
	}
}
