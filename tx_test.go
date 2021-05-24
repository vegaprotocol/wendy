package wendy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTxs(t *testing.T) {
	txs := NewTxs(allTestTxs...)

	t.Run("List", func(t *testing.T) {
		require.Equal(t,
			allTestTxs,
			txs.List(),
		)
	})

	t.Run("ByHash", func(t *testing.T) {
		for _, tx := range allTestTxs {
			require.Equal(t, tx, txs.ByHash(tx.Hash()))
		}

		someHash := Hash{0, 1, 2, 3, 4}
		require.Nil(t, txs.ByHash(someHash))
	})
}

func TestTxsPush(t *testing.T) {
	allTxs := []Tx{testTx0, testTx1}
	txs := NewTxs(allTxs...)

	require.False(t, txs.Push(testTx0), "Push should return False when adding an existing Tx")
	require.Equal(t, allTxs, txs.List())

	require.True(t, txs.Push(testTx2), "Push should return True when adding a new Tx")
	require.Equal(t, append(allTxs, testTx2), txs.List())
}

func TestTxsRemoval(t *testing.T) {
	testCases := []struct {
		name        string
		allTxs      []Tx
		removedTxs  []Tx
		expectedTxs []Tx
	}{
		{
			name:        "Single",
			allTxs:      []Tx{testTx0},
			removedTxs:  []Tx{testTx0},
			expectedTxs: []Tx{},
		},
		{
			name:        "Borders",
			allTxs:      []Tx{testTx0, testTx1, testTx2, testTx3, testTx4, testTx5},
			removedTxs:  []Tx{testTx0, testTx5},
			expectedTxs: []Tx{testTx1, testTx2, testTx3, testTx4},
		},
		{
			name:        "Middle",
			allTxs:      []Tx{testTx0, testTx1, testTx2, testTx3, testTx4, testTx5},
			removedTxs:  []Tx{testTx3},
			expectedTxs: []Tx{testTx0, testTx1, testTx2, testTx4, testTx5},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			txs := NewTxs(test.allTxs...)
			for _, tx := range test.removedTxs {
				txs.RemoveByHash(tx.Hash())
				require.Nil(t, txs.ByHash(tx.Hash()))
			}
			require.Equal(t, test.expectedTxs, txs.List())
		})
	}
}
