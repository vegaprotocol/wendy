package core

import (
	"math/rand"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var debug = func(n *Node, msg string) bool {
	//return !true
	return strings.Contains(msg, "delay")
	// return n.name == "n1"
}

type Network struct {
	nodes map[ID]*Node
}

func NewNetwork(topology map[ID][]ID) *Network {
	nodes := map[ID]*Node{}

	// create new nodes and connect them
	for id, peers := range topology {
		node, ok := nodes[id]
		if !ok {
			pub := NewPubkeyFromID(id)
			node = NewNode(pub).WithDebug(debug)
			nodes[id] = node
		}

		for _, id := range peers {
			peer, ok := nodes[id]
			if !ok {
				pub := NewPubkeyFromID(id)
				peer = NewNode(pub).WithDebug(debug)
				nodes[id] = peer
			}
			node.AddPeer(peer)
		}
	}

	// Collect the validators and update the validator set on every node
	validators := make([]Validator, 0, len(nodes))
	for id := range nodes {
		validators = append(validators, Validator(id))
	}

	net := &Network{
		nodes: nodes,
	}
	for id := range nodes {
		net.Node(id).wendy.UpdateValidatorSet(validators)
	}

	return net
}

func (net *Network) Node(id ID) *Node {
	return net.nodes[id]
}

func (net *Network) Wait() {
	time.Sleep(100 * time.Millisecond)
}

type topology map[ID][]ID

var topologies = map[string]topology{
	"FullyConnected": {
		/*
					 ┌────┐
			   ┌─────┤ 01 ├─────┐
			   │     └──┬─┘     │
			┌──┴─┐      │      ┌┴───┐
			│ 04 ├──────┼──────┤ 02 │
			└──┬─┘      │      └┬───┘
			   │     ┌──┴─┐     │
			   └─────┤ 03 ├─────┘
					 └────┘
		*/
		"01": {"02", "03", "04"},
		"02": {"01", "03", "04"},
		"03": {"01", "02", "04"},
		"04": {"01", "02", "03"},
	},
	"Complex": {
		/*
		   		   ┌────┐
		   		   │ 01 ├──┐
		   		   └─┬──┘  │
		   			 │     │
		   		   ┌─┴──┐  │
		   		   │ 02 │  │
		   		   └─┬──┘  │
		   			 │     │
		 ┌────┐    ┌─┴──┐  │
		 │ 05 ├────┤ 03 ├──┘
		 └────┘    └─┬──┘
		   			 │
		   		   ┌─┴──┐
		   		   │ 04 │
		   		   └────┘
		*/
		"01": {"02", "03"},
		"02": {"01", "03"},
		"03": {"01", "02", "04", "05"},
		"04": {"03"},
		"05": {"03"},
	},
	"Pipeline": {
		/*
			┌────┐  ┌────┐  ┌────┐  ┌────┐
			│ 01 ├──┤ 02 ├──┤ 03 ├──┤ 04 │
			└────┘  └────┘  └────┘  └────┘
		*/
		"01": {"02"},
		"02": {"01", "03"},
		"03": {"02", "04"},
		"04": {"03"},
	},
}

func TestSimulation(t *testing.T) {
	for name, topology := range topologies {
		t.Run(name, func(t *testing.T) {
			t.Run("IsBlocked", func(t *testing.T) {
				testIsBlocked(t, NewNetwork(topology))
			})

			t.Run("WithDelays", func(t *testing.T) {
				testWithDelays(t, NewNetwork(topology))
			})
		})
	}
}

func testIsBlocked(t *testing.T, net *Network) {
	// we use the wg to sync the send/recv operations
	// this way we can know for sure when all messaging is done
	var wg sync.WaitGroup
	for _, node := range net.nodes {
		node.Wg = &wg
	}

	assert.True(t, net.Node("01").wendy.IsBlocked(testTx0), "01")

	for _, tx := range allTestTxs {
		net.Node("01").AddTx(tx)
		wg.Wait()
		assert.False(t, net.Node("01").wendy.IsBlocked(tx), "01")
	}
}

func testWithDelays(t *testing.T, net *Network) {
	var seed int64 = 0x533D
	rand.Seed(seed)

	var wg sync.WaitGroup
	for i := range net.nodes {
		node := net.nodes[i]
		node.Wg = &wg

		// delay receiving messages on 04
		node.RecvCb = func() {
			if node.pub.String() != "04" {
				return
			}

			// delay up to 100ms per message (votes or txs)
			delay := time.Duration(
				rand.Int63n(10),
			)
			time.Sleep(delay * time.Millisecond)
		}
	}

	// generate N transactions
	for _, tx := range allTestTxs {
		net.Node("01").AddTx(tx)
		wg.Wait()
	}

	assert.False(t,
		net.Node("01").wendy.IsBlockedBy(testTx0, testTx1),
	)
}
