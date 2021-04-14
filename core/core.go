package core

import (
	"math"
	"sync"
	"time"
)

const (
	HashLen = 4
	Quorum  = 2 / 3
)

type Hash [HashLen]byte

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

type Wendy struct {
	validators []Validator

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
func (w *Wendy) UpdateValidatorSet(vs []Validator) {
	w.validators = vs
}

// Quorum returns the required number of validators to achieve Quorum.
func (w *Wendy) Quorum() int {
	q := math.Ceil(
		float64(len(w.validators)) * Quorum,
	)
	return int(q)
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

func (w *Wendy) checkForQuorum() {

}
