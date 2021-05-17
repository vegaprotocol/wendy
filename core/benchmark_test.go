package core

import (
	"fmt"
	"testing"
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
		tx := newTestTxStr(
			fmt.Sprintf("tx:%d", seq),
			fmt.Sprintf("hash:%d", seq),
		)
		txs = append(txs, tx)
	}

	// all validators vote on every tx
	for i, tx := range txs {
		for _, v := range vs {
			vote := newVote(Pubkey(v), uint64(i), tx)
			w.AddVote(vote)
		}
	}

	for i := 0; i < b.N; i++ {
		w.IsBlockedBy(txs[0], txs[1])
	}
}
