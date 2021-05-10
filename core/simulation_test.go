package core

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var debug = func(n *Node, msg string) bool {
	//return !true
	return strings.Contains(msg, "IF:")
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
	for _, n := range net.nodes {
		n.Wait()
	}
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
}

func TestSimulation(t *testing.T) {
	for name, topology := range topologies {
		t.Run(name, func(t *testing.T) {
			net := NewNetwork(topology)
			testSimulation(t, net)
		})
	}
}

func testSimulation(t *testing.T, net *Network) {
	assert.True(t, net.Node("n1").wendy.IsBlocked(testTx0), "n1")

	net.Node("n1").AddTx(testTx0)
	net.Wait()
	assert.False(t, net.Node("n1").wendy.IsBlocked(testTx0), "n1")
}
