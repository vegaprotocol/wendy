package wendy

import (
	"fmt"
)

var _ Tx = &SimpleTx{}

type SimpleTx struct {
	bytes []byte
	hash  []byte
	label string
}

func NewSimpleTx(bytes, hash string) *SimpleTx {
	return &SimpleTx{bytes: []byte(bytes), hash: []byte(hash)}
}

func (tx *SimpleTx) Bytes() []byte { return tx.bytes }
func (tx *SimpleTx) Hash() Hash {
	var hash Hash
	copy(hash[:], tx.hash)
	return hash
}
func (tx *SimpleTx) Label() string { return tx.label }

func (tx *SimpleTx) withLabel(l string) *SimpleTx {
	tx.label = l
	return tx
}

func (tx *SimpleTx) String() string {
	return fmt.Sprintf("%s (hash:%s)", string(tx.bytes), string(tx.hash))
}

// Txs keeps track of one or more Tx in order (.List()) but it also provide a
// fast way to locate them by Hash (.ByHash).
// Txs keeps unique transactions, and it discards added Txs that are duplicated.
// Txs is not safe for concurrent access and the caller should protect it from
// races.
type Txs struct {
	list   []Tx
	byHash map[Hash]int
}

// NewTxs returns a new Txs and optionally initializes with Txs.
func NewTxs(list ...Tx) *Txs {
	txs := &Txs{}
	txs.reset()

	for _, tx := range list {
		txs.Push(tx)
	}

	return txs
}

func (txs *Txs) reset() {
	txs.list = []Tx{}
	txs.byHash = make(map[Hash]int)
}

// Push pushes a new tx to the list and returns true if the above tx hasn't been
// added before. If the tx has been already added, it's not added again and it
// returns false.
func (txs *Txs) Push(tx Tx) bool {
	hash := tx.Hash()
	if _, ok := txs.byHash[hash]; ok {
		return false
	}

	txs.list = append(txs.list, tx)
	txs.byHash[hash] = len(txs.list) - 1
	return true
}

// List returns the underlying slice of pushed transactions.
func (txs *Txs) List() []Tx {
	return txs.list
}

// ByHash returns a Tx given its Hash, if the Tx is not found, it returns nil.
// ByHash provides a fast way to retrieve a Tx without iterating the whole set.
func (txs *Txs) ByHash(hash Hash) Tx {
	pos, ok := txs.byHash[hash]
	if !ok {
		return nil
	}
	return txs.list[pos]
}

// RemoveByHash removes a Tx by its hash, it returns true if the Tx has been
// effectively removed and false if the Tx was not found.
func (txs *Txs) RemoveByHash(hash Hash) bool {
	pos, ok := txs.byHash[hash]
	if !ok {
		return false
	}

	if len(txs.list) == 1 {
		txs.reset()
		return true
	}

	// re-index items in front by decreasing their position
	for _, tx := range txs.list[pos+1:] {
		txs.byHash[tx.Hash()]--
	}

	// remove the item from the list and the index
	txs.list = append(
		txs.list[:pos],
		txs.list[pos+1:]...,
	)
	delete(txs.byHash, hash)
	return true
}
