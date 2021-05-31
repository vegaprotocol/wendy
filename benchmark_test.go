package wendy

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func BenchmarkIsBlockedBy2(b *testing.B)    { benchmarkIsBlockedBy(b, 2) }
func BenchmarkIsBlockedBy10(b *testing.B)   { benchmarkIsBlockedBy(b, 10) }
func BenchmarkIsBlockedBy50(b *testing.B)   { benchmarkIsBlockedBy(b, 50) }
func BenchmarkIsBlockedBy100(b *testing.B)  { benchmarkIsBlockedBy(b, 100) }
func BenchmarkIsBlockedBy500(b *testing.B)  { benchmarkIsBlockedBy(b, 500) }
func BenchmarkIsBlockedBy1000(b *testing.B) { benchmarkIsBlockedBy(b, 1000) }

func benchmarkIsBlockedBy(b *testing.B, n int) {
	w := New()
	vs := []Validator{
		pub0.Bytes(), pub1.Bytes(), pub2.Bytes(), pub3.Bytes(),
	}
	w.UpdateValidatorSet(vs)

	// build all the txs
	var txs = make([]Tx, 0, n)
	for seq := 0; seq < n; seq++ {
		tx := NewSimpleTx(
			fmt.Sprintf("tx:%d", seq),
			fmt.Sprintf("hash:%d", seq),
		)
		txs = append(txs, tx)
	}

	// all validators vote on every tx
	for _, v := range vs {
		var prevVote *Vote
		for i, tx := range txs {
			vote := NewVote(Pubkey(v), uint64(i), tx)
			if pv := prevVote; pv != nil {
				vote.WithPrevHash(pv.Hash())
			}
			prevVote = vote
			_, err := w.AddVote(vote)
			require.NoError(b, err)
		}
	}

	for i := 0; i < b.N; i++ {
		w.IsBlockedBy(txs[0], txs[1])
	}
}
