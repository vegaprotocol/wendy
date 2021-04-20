package core

import (
	"math"
	"sync"
	"time"
)

const (
	HashLen = 32
)

var (
	Quorum = float64(2) / 3
)

type Hash [HashLen]byte

func (h Hash) String() string { return string(h[:]) }

type ID string

type Tx interface {
	Bytes() []byte
	Hash() Hash
}

type Validator string

type Vote struct {
	Seq    uint64
	TxHash Hash
	Time   time.Time
	// TODO: signature
}

func newVote(seq uint64, tx Tx) *Vote {
	return &Vote{Seq: seq, TxHash: tx.Hash(), Time: time.Now()}
}

type Wendy struct {
	validators []Validator
	quorum     int // quorum gets updated every time the validator set is updated.

	txsMtx sync.RWMutex
	txs    map[Hash]Tx

	votesMtx sync.RWMutex
	votes    map[Hash]*Vote
	senders  map[ID]*Sender
}

func New() *Wendy {
	return &Wendy{
		txs:     make(map[Hash]Tx),
		votes:   make(map[Hash]*Vote),
		senders: make(map[ID]*Sender),
	}
}

// UpdateValidatorSet updates the list of validators in the consensus.
// Updating the validator set might affect the return value of Quorum().
// Upon updating the senders that are not in the new validator set are removed.
func (w *Wendy) UpdateValidatorSet(vs []Validator) {
	w.validators = vs

	q := math.Floor(
		float64(len(vs))*Quorum,
	) + 1

	w.quorum = int(q)

	senders := make(map[ID]*Sender)
	// keep all the senders we already have and create new one if not present
	// those old senders that are not part of the new set will be discarded.
	for _, v := range vs {
		key := ID(v)
		if s, ok := w.senders[key]; ok {
			senders[key] = s
		} else {
			senders[key] = NewSender(key)
		}
	}
	w.senders = senders
}

func (w *Wendy) Quorum() int {
	return w.quorum
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
// NOTE: This function is safe for concurrent access.
func (w *Wendy) AddVote(senderID ID, v *Vote) {
	w.votesMtx.Lock()
	defer w.votesMtx.Unlock()

	// Register the vote on the sender
	sender, ok := w.senders[senderID]
	if !ok {
		sender = NewSender(senderID)
		w.senders[senderID] = sender
	}
	sender.AddVote(v)

	// Register the vote based on its tx.Hash
	w.votes[v.TxHash] = v
}

// VoteByTxHash returns a vote given its tx.Hash
// Returns nil if the vote hasn't been seen.
// NOTE: This function is safe for concurrent access.
func (w *Wendy) VoteByTxHash(hash Hash) *Vote {
	w.votesMtx.RLock()
	defer w.votesMtx.RUnlock()
	return w.votes[hash]
}

// hasQuorum evaluates fn for every register sender.
// It returns true if fn returned true at least w.quorum times.
func (w *Wendy) hasQuorum(fn func(s *Sender) bool) bool {
	var votes int
	for _, s := range w.senders {
		if ok := fn(s); ok {
			votes++
			if votes == w.quorum {
				return true
			}
		}

	}
	return false
}

// IsBlockedBy determines if tx2 might have priority over tx1.
// We say that tx1 is NOT bloked by tx2 if there are t+1 votes reporting tx1
// before tx2.
func (w *Wendy) IsBlockedBy(tx1, tx2 Tx) bool {
	// tx1 is BlockedBy tx2 if we couldn't find agreement that tx1 is before
	// tx2
	return !w.hasQuorum(func(s *Sender) bool {
		return s.Before(tx1, tx2)
	})
}

// IsBlocked identifies if it is pssible that a so-far-unknown transaction
// might be scheduled with priority to tx.
func (w *Wendy) IsBlocked(tx Tx) bool {
	// tx is Blocked if there if we couldn't find agreement that tx has been
	// seen.
	return !w.hasQuorum(func(s *Sender) bool {
		return s.Seen(tx)
	})
}
