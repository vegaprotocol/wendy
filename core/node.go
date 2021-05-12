package core

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var (
	wg sync.WaitGroup
)

type nodeVote struct {
	vote  *Vote
	from  *Node
	owner *Node
}

func (nv *nodeVote) String() string {
	return fmt.Sprintf("from=%s, vote=(%s)", nv.from.name, nv.vote)
}

type nodeTx struct {
	tx   Tx
	from *Node
}

func (nt *nodeTx) String() string {
	return fmt.Sprintf("from=%s, tx=(%s)", nt.from.name, nt.tx)
}

type NodeDebugFn func(*Node, string) bool

// Node is used in the simulations and represents a node in the network.
// A Node can be connected to other peers. Connections are unidirectional,
// meaning that if NodeA is connected to NodeB, NodeB will receive messages
// from NodeA.
type Node struct {
	wendy *Wendy
	name  ID
	peers []*Node

	seq uint64

	debug NodeDebugFn

	txs   chan nodeTx
	votes chan nodeVote

	Wg     *sync.WaitGroup
	SendCb func() // SendCb is called before sending a msg iff not nil.
	RecvCb func() // RecvCb is called after handling a msg iff not nil.
}

func NewNode(name ID, peers ...*Node) *Node {
	node := &Node{
		wendy: New(),
		name:  name,
		peers: peers,

		txs:   make(chan nodeTx),
		votes: make(chan nodeVote),
	}

	go node.recvMsgs()

	return node
}

func (n *Node) WithDebug(fn NodeDebugFn) *Node {
	n.debug = fn
	return n
}

func (n *Node) AddPeer(peer *Node) {
	if peer.name == n.name {
		panic("can't add myself as peer")
	}

	n.peers = append(n.peers, peer)
}

func (n *Node) AddPeers(peers ...*Node) {
	for _, peer := range peers {
		n.AddPeer(peer)
	}
}

func (n *Node) nextVote(tx Tx) *Vote {
	seq := atomic.AddUint64(&n.seq, 1)
	return newVote(n.name, seq-1, tx)
}

func (n *Node) AddTx(tx Tx) {
	msg := &nodeTx{from: n, tx: tx}
	n.handleTx(msg)
}

func (n *Node) log(msg string, args ...interface{}) {
	line := fmt.Sprintf("[%s] %s\n", n.name, fmt.Sprintf(msg, args...))
	if n.debug(n, line) {
		fmt.Printf(line)
	}
}

func (n *Node) handleTx(msg *nodeTx) {
	if isNew := n.wendy.AddTx(msg.tx); !isNew {
		return
	}
	// broadcast to tx and myvote to peers
	from := msg.from
	msg = &nodeTx{tx: msg.tx, from: n}
	for _, peer := range n.peers {
		// avoid sending the tx back to the sender
		if from.name == peer.name {
			continue
		}

		n.send(peer, msg)
	}
	myVote := &nodeVote{from: n, vote: n.nextVote(msg.tx)}
	n.handleVote(myVote)
}

func (n *Node) handleVote(msg *nodeVote) {
	if isNew := n.wendy.AddVote(msg.vote); !isNew {
		return
	}

	from := msg.from
	msg = &nodeVote{from: n, vote: msg.vote}
	for _, peer := range n.peers {
		if from.name == peer.name || msg.vote.Pubkey == peer.name {
			continue
		}

		n.send(peer, msg)
	}
}

// send sends a message to a given peer asynchronously (inside a goroutine).
// It increments the wg and calls SendCb if they are set.
func (n *Node) send(peer *Node, i interface{}) {
	if wg := n.Wg; wg != nil {
		wg.Add(1)
	}

	go func() {
		if fn := n.SendCb; fn != nil {
			fn()
		}

		switch msg := i.(type) {
		case *nodeTx:
			peer.txs <- *msg
		case *nodeVote:
			peer.votes <- *msg
		}
	}()
}

func (n *Node) recvMsgs() {
	var handler func()

	for {
		select {
		case tx := <-n.txs:
			handler = func() { n.handleTx(&tx) }
		case vote := <-n.votes:
			handler = func() { n.handleVote(&vote) }
		}

		if fn := n.RecvCb; fn != nil {
			fn()
		}

		handler()

		if wg := n.Wg; wg != nil {
			wg.Done()
		}
	}
}
