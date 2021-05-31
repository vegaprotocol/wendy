package simulation

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/vegaprotocol/wendy"
)

type nodeVote struct {
	vote *wendy.Vote
	from *Node
}

func (nv *nodeVote) String() string {
	return fmt.Sprintf("from=%s, vote=(%s)", nv.from.pub.String(), nv.vote)
}

type nodeTx struct {
	tx   wendy.Tx
	from *Node
}

func (nt *nodeTx) String() string {
	return fmt.Sprintf("from=%s, tx=(%s)", nt.from.pub.String(), nt.tx)
}

type NodeDebugFn func(*Node, string) bool

// Node is used in the simulations and represents a node in the network.
// A Node can be connected to other peers. Connections are unidirectional,
// meaning that if NodeA is connected to NodeB, NodeB will receive messages
// from NodeA.
type Node struct {
	wendy    *wendy.Wendy
	pub      wendy.Pubkey
	peers    []*Node
	lastVote *wendy.Vote

	seq uint64

	debug NodeDebugFn

	txs   chan nodeTx
	votes chan nodeVote

	Wg     *sync.WaitGroup
	SendCb func() // SendCb is called before sending a msg iff not nil.
	RecvCb func() // RecvCb is called after handling a msg iff not nil.
}

func NewNode(pub wendy.Pubkey, peers ...*Node) *Node {
	node := &Node{
		wendy: wendy.New(),
		pub:   pub,
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
	if peer.pub.String() == n.pub.String() {
		panic("can't add myself as peer")
	}

	n.peers = append(n.peers, peer)
}

func (n *Node) AddPeers(peers ...*Node) {
	for _, peer := range peers {
		n.AddPeer(peer)
	}
}

func (n *Node) nextVote(tx wendy.Tx) *wendy.Vote {
	seq := atomic.AddUint64(&n.seq, 1)
	vote := wendy.NewVote(n.pub, seq-1, tx)
	if last := n.lastVote; last != nil {
		vote.PrevHash = last.Hash()
	}
	n.lastVote = vote
	return vote
}

func (n *Node) AddTx(tx wendy.Tx) {
	msg := &nodeTx{from: n, tx: tx}
	n.handleTx(msg)
}

func (n *Node) Log(msg string, args ...interface{}) {
	line := fmt.Sprintf("[%s] %s\n", n.pub.String(), fmt.Sprintf(msg, args...))
	if n.debug(n, line) {
		fmt.Print(line)
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
		if from.pub.String() == peer.pub.String() {
			continue
		}

		n.send(peer, msg)
	}
	myVote := &nodeVote{from: n, vote: n.nextVote(msg.tx)}
	n.handleVote(myVote)
}

func (n *Node) handleVote(msg *nodeVote) error {
	isNew, err := n.wendy.AddVote(msg.vote)
	if err != nil {
		return err
	}

	if !isNew {
		return nil
	}

	from := msg.from
	msg = &nodeVote{from: n, vote: msg.vote}
	for _, peer := range n.peers {
		if from.pub.String() == peer.pub.String() || msg.vote.Pubkey.String() == peer.pub.String() {
			continue
		}

		n.send(peer, msg)
	}
	return nil
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
