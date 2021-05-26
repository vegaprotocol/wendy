package list

import (
	"container/list"
	"fmt"
	"strings"
)

type Element = list.Element

// List is an augmentation of a container/list that provides convenience
// functions.
type List struct {
	*list.List
}

// New returns a new List
func New() *List {
	return &List{
		List: list.New(),
	}
}

// FilterFunc is a user provided function to filter elements on a list.
type FilterFunc func(e *Element) bool

// Each iterate over a List executing fn on every item.
// Each will stop the iteration if fn returns false.
func (l *List) Each(fn FilterFunc, opts ...Option) {
	if Options(opts).Backward() {
		l.eachBackward(fn)
	} else {
		l.eachForward(fn)
	}
}

func (l *List) eachBackward(fn FilterFunc) {
	var prev *list.Element
	for e := l.Back(); e != nil; e = prev {
		prev = e.Prev()
		if cont := fn(e); !cont {
			return
		}
	}
}

func (l *List) eachForward(fn FilterFunc) {
	var next *list.Element
	for e := l.Front(); e != nil; e = next {
		next = e.Next()
		if cont := fn(e); !cont {
			return
		}
	}
}

// First returns the first element in the list matching FilterFunc.
func (l *List) First(fn FilterFunc, opts ...Option) *list.Element {
	var found *list.Element
	l.Each(func(e *list.Element) bool {
		if fn(e) {
			found = e
		}
		return found == nil
	}, opts...)

	return found
}

// Discard removes the first item matching the FilterFunc from the list.
func (l *List) DiscardFirst(fn FilterFunc, opts ...Option) bool {
	e := l.First(fn, opts...)
	if e != nil {
		l.List.Remove(e)
		return true
	}
	return false
}

// Discard removes all items matching the FilterFunc from the list.
func (l *List) Discard(fn FilterFunc, opts ...Option) int {
	var count int
	l.Each(func(e *list.Element) bool {
		if match := fn(e); match {
			count++
			l.List.Remove(e)
		}
		return true
	}, opts...)
	return count
}

// All returns a new list containing all the matching elements given the
// FilterFunc fn.
func (l *List) All(fn FilterFunc, opts ...Option) *List {
	var all = New()
	l.Each(func(e *list.Element) bool {
		if fn(e) {
			all.PushBack(e.Value)
		}
		return true
	}, opts...)
	return all
}

// String returns the string representation of the List
// It calls .String() on items if defined.
func (l *List) String() string {
	var items = make([]string, 0, l.Len())
	l.Each(func(e *Element) bool {
		items = append(items, fmt.Sprintf("%s", e.Value))
		return true
	})

	return fmt.Sprintf("[\n%s\n]", strings.Join(items, ",\n"))
}
