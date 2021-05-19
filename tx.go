package wendy

import "fmt"

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
