package list

import (
	"container/list"
	"testing"

	"github.com/stretchr/testify/require"
)

var testFilter = func(e *list.Element) bool {
	return e.Value.(int) > 3
}

var testItems = []int{1, 2, 3, 4, 5}

func TestList(t *testing.T) {
	tests := map[string]func(t *testing.T, list *List){
		"First":        testListFirst,
		"All":          testListAll,
		"DiscardFirst": testListDiscardFirst,
		"Discard":      testListDiscard,
	}

	for test, fn := range tests {
		l := New()
		for _, item := range testItems {
			l.PushBack(item)
		}

		t.Run(test, func(t *testing.T) {
			fn(t, l)
		})
	}
}

func testListFirst(t *testing.T, l *List) {
	found := l.First(testFilter)

	require.Equal(t, 4, found.Value)
}

func testListAll(t *testing.T, l *List) {
	all := l.All(testFilter)

	require.Equal(t, 2, all.Len())
	require.Equal(t, 4, all.Front().Value.(int))
}

func testListDiscardFirst(t *testing.T, l *List) {
	initialLen := l.Len()

	l.DiscardFirst(testFilter)
	require.Equal(t, initialLen-1, l.Len())

	found := l.First(testFilter)
	require.Equal(t, 5, found.Value)
}

func testListDiscard(t *testing.T, l *List) {
	initialLen := l.Len()

	l.Discard(testFilter)
	require.Equal(t, initialLen-2, l.Len())

	found := l.First(testFilter)
	require.Nil(t, found)

	require.Equal(t, 1, l.Front().Value)
	require.Equal(t, 3, l.Back().Value)
}
