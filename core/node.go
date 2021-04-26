package core

import (
	"fmt"
	"sync/atomic"
)

type nodeVote struct {
	vote  *Vote
	from  ID
	owner ID
}

func (nv *nodeVote) String() string {
	return fmt.Sprintf("from=%s, vote=(%s)", nv.from, nv.vote)
}

type nodeTx struct {
	tx   Tx
	from ID
}

func (nt *nodeTx) String() string {
	return fmt.Sprintf("from=%s, tx=(%s)", nt.from, nt.tx)
}

type debugFn func(*Node) bool

type Node struct {
	wendy *Wendy
	name  ID
	peers []*Node

	seq uint64

	debug func(n *Node) bool
	cheat bool
}

func NewNode(name ID, peers ...*Node) *Node {
	node := &Node{
		wendy: New(),
		name:  name,
		peers: peers,
	}

	return node
}

func (n *Node) WithDebug(fn debugFn) *Node {
	n.debug = fn
	return n
}

func (n *Node) WithCheating(v bool) *Node {
	n.cheat = v
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
	msg := &nodeTx{from: n.name, tx: tx}
	n.handleTx(msg)
}

func (n *Node) log(msg string, args ...interface{}) {
	if n.debug(n) {
		fmt.Printf("[%s] %s\n", n.name, fmt.Sprintf(msg, args...))
	}
}

func (n *Node) handleTx(msg *nodeTx) {
	if isNew := n.wendy.AddTx(msg.tx); !isNew {
		return
	}
	// broadcast to tx and myvote to peers
	from := msg.from
	msg.from = n.name
	for _, peer := range n.peers {
		// avoid sending the tx back to the sender
		if from == peer.name {
			continue
		}

		n.log("tx -> %s", peer.name)
		peer.handleTx(msg)
	}
	myVote := &nodeVote{from: n.name, vote: n.nextVote(msg.tx)}
	n.handleVote(myVote)
}

func (n *Node) handleVote(msg *nodeVote) {
	if isNew := n.wendy.AddVote(msg.vote); !isNew {
		return
	}

	from := msg.from
	msg.from = n.name
	for _, peer := range n.peers {
		if from == peer.name || msg.vote.Pubkey == peer.name {
			continue
		}

		n.log("vote(%s) -> %s", msg.vote.Pubkey, peer.name)
		peer.handleVote(msg)
	}

}
