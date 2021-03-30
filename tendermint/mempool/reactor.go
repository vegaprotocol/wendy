package main

import (
	"fmt"
	"os"
	"time"

	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/p2p/conn"
)

const (
	Channel = byte(0x80)
)

type Reactor struct {
	p2p.BaseReactor
}

func NewReactor() *Reactor {
	r := &Reactor{}
	r.BaseReactor = *p2p.NewBaseReactor("WendyReactor", r)
	return r
}

// GetChannels returns the channels used by this reactor.
// Satisfies the p2p.Reactor interface.
func (r *Reactor) GetChannels() []*conn.ChannelDescriptor {
	return []*conn.ChannelDescriptor{
		{ID: Channel, Priority: 2},
	}
}

func (r *Reactor) InitPeer(peer p2p.Peer) p2p.Peer {
	peer.SetLogger(
		log.NewTMLogger(os.Stdout),
	)
	return peer
}

// AddPeer is used to start goroutines communicating with the peer. It is
// called after InitPeer.
// Satisfies the p2p.Reactor interface.
func (r *Reactor) AddPeer(peer p2p.Peer) {
	go r.broadcastTxs(peer)
}

func (r *Reactor) broadcastTxs(peer p2p.Peer) {
	for {
		if !r.IsRunning() || !peer.IsRunning() {
			fmt.Printf("Peer not running!")
			return
		}

		select {
		case <-time.After(1 * time.Second):
			fmt.Printf("Sending:\n")
			peer.Send(Channel, []byte("hello"))
		}
	}
}

// Receive handles messages coming to the reactor. This is the counter-part of
// peer.Send.
// Satisfies the p2p.Reactor interface.
func (r *Reactor) Receive(chID byte, peer p2p.Peer, msgBytes []byte) {
	fmt.Printf("!!!!!!! msgBytes = %+v\n", msgBytes)
}

func (r *Reactor) RemovePeer(peer p2p.Peer, reason interface{}) {
}
