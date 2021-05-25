package wendy

import (
	"crypto/ed25519"
	"testing"

	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/assert"
)

/*
 * This file defines variour reusable testing variables as well as some type test.
 * For some tests it uses golden files via goldie package.
 */

func newRandPubkey() Pubkey {
	pub, _, err := ed25519.GenerateKey(Rand)
	if err != nil {
		panic(err)
	}

	return Pubkey(pub)
}

var (
	pub0 = newRandPubkey()
	pub1 = newRandPubkey()
	pub2 = newRandPubkey()
	pub3 = newRandPubkey()
)

var (
	testTx0    = NewSimpleTx("tx0", "h0")
	testTx1    = NewSimpleTx("tx1", "h1")
	testTx2    = NewSimpleTx("tx2", "h2")
	testTx3    = NewSimpleTx("tx3", "h3")
	testTx4    = NewSimpleTx("tx4", "h4")
	testTx5    = NewSimpleTx("tx5", "h5")
	allTestTxs = []Tx{testTx0, testTx1, testTx2, testTx3, testTx4, testTx5}
)

var (
	testVote0    = &Vote{Pubkey: pub0, Seq: 0, TxHash: testTx0.Hash()}
	testVote1    = &Vote{Pubkey: pub0, Seq: 1, TxHash: testTx1.Hash(), PrevHash: testVote0.Hash()}
	testVote2    = &Vote{Pubkey: pub0, Seq: 2, TxHash: testTx2.Hash(), PrevHash: testVote1.Hash()}
	testVote3    = &Vote{Pubkey: pub0, Seq: 3, TxHash: testTx3.Hash(), PrevHash: testVote2.Hash()}
	testVote4    = &Vote{Pubkey: pub0, Seq: 4, TxHash: testTx4.Hash(), PrevHash: testVote3.Hash()}
	allTestVotes = []*Vote{testVote0, testVote1, testVote2, testVote3, testVote4}
)

// Note that Pubkey (pub0) matches the one used on testVote<N> on purpose
var newTestPeer = func() *Peer { return NewPeer(pub0) }

// TestVotes uses golden files via goldie package.  In order to update
// the golden files, run the tests using -update flag, i.e: go test -update
// This test ensures that the hashing function or digest function did not change.
func TestVotes(t *testing.T) {
	hash0 := testVote0.Hash().String()
	hash1 := testVote1.Hash().String()
	hash2 := testVote2.Hash().String()
	hash3 := testVote3.Hash().String()
	hash4 := testVote4.Hash().String()

	t.Run("Hashes", func(t *testing.T) {
		g := goldie.New(t)
		g.Assert(t, "testVote0.hash", []byte(hash0))
		g.Assert(t, "testVote1.hash", []byte(hash1))
		g.Assert(t, "testVote2.hash", []byte(hash2))
		g.Assert(t, "testVote3.hash", []byte(hash3))
		g.Assert(t, "testVote4.hash", []byte(hash4))
	})

	t.Run("PrevHash", func(t *testing.T) {
		assert.Equal(t, testVote0.Hash(), testVote1.PrevHash)
		assert.Equal(t, testVote1.Hash(), testVote2.PrevHash)
		assert.Equal(t, testVote2.Hash(), testVote3.PrevHash)
		assert.Equal(t, testVote3.Hash(), testVote4.PrevHash)
	})
}
