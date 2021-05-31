package wendy

import (
	"math"
	"sort"
	"sync"
)

// Wendy is the root of the Wendy fairness implementation. It holds a set of
// peers and acts as a proxy to them. Wendy keeps track of all Peers's state
// and aggregates them in order to do vote counting.
//
// Invoking Wendy methods is thread safe.
type Wendy struct {
	validators []Validator
	quorum     int // quorum gets updated every time the validator set is updated.

	txsMtx sync.RWMutex
	txs    *Txs

	peersMtx sync.RWMutex
	votes    map[Hash]*Vote
	peers    map[ID]*Peer
}

// New returns a new Wendy instance.
// Normally a mempool should hold only one instance.
func New() *Wendy {
	return &Wendy{
		txs:   NewTxs(),
		votes: make(map[Hash]*Vote),
		peers: make(map[ID]*Peer),
	}
}

// UpdateValidatorSet updates the list of validators in the consensus.
// Updating the validator set might affect the value of the Quorum field.
// Upon updating the peers that are not in the new validator set are removed.
func (w *Wendy) UpdateValidatorSet(vs []Validator) {
	w.validators = vs

	q := math.Floor(
		float64(len(vs))*Quorum,
	) + 1
	w.quorum = int(q)

	w.peersMtx.Lock()
	defer w.peersMtx.Unlock()
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
func (w *Wendy) AddTx(tx Tx) bool {
	w.txsMtx.Lock()
	defer w.txsMtx.Unlock()

	return w.txs.Push(tx)
}

// AddVote adds a vote to the list of votes.
// Votes are positioned given it's sequence number.
// AddVote returns alse if the vote was already added.
func (w *Wendy) AddVote(v *Vote) (bool, error) {
	w.peersMtx.Lock()
	defer w.peersMtx.Unlock()

	key := ID(v.Pubkey.String())
	// Register the vote on the peer
	peer, ok := w.peers[key]
	if !ok {
		pub := NewPubkeyFromID(key)
		peer = NewPeer(pub)
		w.peers[key] = peer
	}

	ok, err := peer.AddVote(v)
	if err != nil {
		return false, err
	}

	// Register the vote based on its tx.Hash
	w.votes[v.TxHash] = v
	return ok, nil
}

func (w *Wendy) AddVotes(vs ...*Vote) error {
	for _, v := range vs {
		if _, err := w.AddVote(v); err != nil {
			return err
		}
	}
	return nil
}

// CommitBlock iterate over the block's Txs set and remove them from Wendy's
// internal state.
// Txs present on block were probbaly added in the past via AddTx().
func (w *Wendy) CommitBlock(block Block) {
	w.peersMtx.Lock()
	defer w.peersMtx.Unlock()

	for _, peer := range w.peers {
		peer.UpdateTxSet(block.Txs...)
	}
}

// VoteByTxHash returns a vote given its tx.Hash
// Returns nil if the vote hasn't been seen.
func (w *Wendy) VoteByTxHash(hash Hash) *Vote {
	w.peersMtx.RLock()
	defer w.peersMtx.RUnlock()
	return w.votes[hash]
}

// hasQuorum evaluates fn for every registered peer.
// It returns true if fn returned true at least w.Quorum() times.
// NOTE: This function is safe for concurrent access.
func (w *Wendy) hasQuorum(fn func(*Peer) bool) bool {
	w.peersMtx.RLock()
	defer w.peersMtx.RUnlock()

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

// AddBlock will clean all the added txs (via AddTx).
// Once a block has been added, all the txs will be removed from Wendy, thus
// new blocks (NewBlock) won't return them anymore.
func (w *Wendy) AddBlock(block *Block) {
	w.txsMtx.Lock()
	defer w.txsMtx.Unlock()
	for _, tx := range block.Txs {
		w.txs.RemoveByHash(tx.Hash())
	}

	w.peersMtx.Lock()
	defer w.peersMtx.Unlock()
	for _, peer := range w.peers {
		peer.UpdateTxSet(block.Txs...)
	}
}

// NewBlockOptions are options that control the behaviour of NewBlock method.
type NewBlockOptions struct {
	// TxLimit limits the maximum number of Txs that a produced block might
	// contain.
	TxLimit int

	// MaxBlockSize limits the maximum size of a block.
	// MaxBlockSize is set in bytes and is computed as the sum of all
	// `len(tx.Bytes())`.
	// If a tx makes exceed the BlockSize, it is removed from the block and the
	// function returns.
	// NewBlock will not try to optimize for space.
	MaxBlockSize int

	// AddBlock flag determines if the newly created block should be also added.
	AddBlock bool
}

// NewBlock produces a potential block given the computed BlockingSet.
// The new block will contain a set of Txs that need to go all together in the
// same block.
func (w *Wendy) NewBlock() *Block {
	return w.NewBlockWithOptions(
		NewBlockOptions{},
	)
}

// NewBlockWithOptions produces a potential block given the computed
// BlockingSet and a set of options.
// The new block will contain a set of Txs that need to go all
// together in the same block.
func (w *Wendy) NewBlockWithOptions(opts NewBlockOptions) *Block {
	w.txsMtx.RLock()
	defer w.txsMtx.RUnlock()

	var (
		// these are used to keep track of the different limits
		size int
	)

	txs := NewTxs()
	set := w.BlockingSet()

	for _, tx := range w.txs.List() {
		list := set[tx.Hash()]
		for _, tx := range list {
			if limit := opts.TxLimit; limit > 0 {
				if len(txs.List()) == limit {
					break
				}
			}

			if max := opts.MaxBlockSize; max > 0 {
				size += len(tx.Bytes())
				if size > max {
					break
				}
			}

			txs.Push(tx)
		}
	}

	block := &Block{
		Txs: txs.List(),
	}

	if opts.AddBlock {
		w.AddBlock(block)
	}

	return block
}

// BlockingSet returns a list of blocking Txs for all the currently seen Txs.
func (w *Wendy) BlockingSet() BlockingSet {
	txs := w.txs.List()

	// Build the dependency matrix for all Txs
	var matrix [][]bool = make([][]bool, len(txs))
	for i := range matrix {
		matrix[i] = make([]bool, len(txs))
	}
	for i, tx1 := range w.txs.List() {
		for j, tx2 := range txs {
			matrix[i][j] = w.IsBlockedBy(tx1, tx2)
		}
	}

	set := BlockingSet{}
	for i, tx := range txs {
		blockers := []Tx{}
		deps := make(map[int]struct{})
		recompute(matrix, i, deps)

		var keys sort.IntSlice = make([]int, 0, len(deps))
		for i := range deps {
			keys = append(keys, i)
		}
		sort.Sort(keys)

		for txIndex := range keys {
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
