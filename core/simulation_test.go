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
			node = NewNode(id).WithDebug(debug)
			nodes[id] = node
		}

		for _, id := range peers {
			peer, ok := nodes[id]
			if !ok {
				peer = NewNode(id).WithDebug(debug)
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
			   ┌─────┤ n1 ├─────┐
			   │     └──┬─┘     │
			┌──┴─┐      │      ┌┴───┐
			│ n4 ├──────┼──────┤ n2 │
			└──┬─┘      │      └┬───┘
			   │     ┌──┴─┐     │
			   └─────┤ n3 ├─────┘
					 └────┘
		*/
		"n1": {"n2", "n3", "n4"},
		"n2": {"n1", "n3", "n4"},
		"n3": {"n1", "n2", "n4"},
		"n4": {"n1", "n2", "n3"},
	},
	"Complex": {
		/*
		   		   ┌────┐
		   		   │ n1 ├──┐
		   		   └─┬──┘  │
		   			 │     │
		   		   ┌─┴──┐  │
		   		   │ n2 │  │
		   		   └─┬──┘  │
		   			 │     │
		 ┌────┐    ┌─┴──┐  │
		 │ n5 ├────┤ n3 ├──┘
		 └────┘    └─┬──┘
		   			 │
		   		   ┌─┴──┐
		   		   │ n4 │
		   		   └────┘
		*/
		"n1": {"n2", "n3"},
		"n2": {"n1", "n3"},
		"n3": {"n1", "n2", "n4", "n5"},
		"n4": {"n3"},
		"n5": {"n3"},
	},
	"Pipeline": {
		/*
			┌────┐  ┌────┐  ┌────┐  ┌────┐
			│ n1 ├──┤ n2 ├──┤ n3 ├──┤ n4 │
			└────┘  └────┘  └────┘  └────┘
		*/
		"n1": {"n2"},
		"n2": {"n1", "n3"},
		"n3": {"n2", "n4"},
		"n4": {"n3"},
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

	assert.True(t, net.Node("n1").wendy.IsBlocked(testTx0), "n1")

	for _, tx := range allTestTxs {
		net.Node("n1").AddTx(tx)
		wg.Wait()
		assert.False(t, net.Node("n1").wendy.IsBlocked(tx), "n1")
	}
}

func testWithDelays(t *testing.T, net *Network) {
	var seed int64 = 0x533D
	rand.Seed(seed)

	var wg sync.WaitGroup
	for i := range net.nodes {
		node := net.nodes[i]
		node.Wg = &wg

		// delay receiving messages on node4
		node.RecvCb = func() {
			if node.name != "n4" {
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
		net.Node("n1").AddTx(tx)
		wg.Wait()
	}

	assert.False(t,
		net.Node("n1").wendy.IsBlockedBy(testTx0, testTx1),
	)
}
