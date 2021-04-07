package wendy

import (
	"log"

	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/p2p/conn"
	"github.com/tendermint/tendermint/types"

	protowendy "code.vegaprotocol.io/wendy/proto/wendy"
)

const (
	WendyChannel = byte(0x99)
)

type Reactor struct {
	p2p.BaseReactor
	txChan chan (types.Tx)
}

func NewReactor() *Reactor {
	r := &Reactor{
		txChan: make(chan types.Tx),
	}
	r.BaseReactor = *p2p.NewBaseReactor("Wendy", r)
	return r
}

// OnNewTx is a handler for a new incoming Tx.
// Since it is designed to react upon new Tx on the mempool, the signature
// satisfies the mempool.NotifyFunc.
func (r *Reactor) OnNewTx(tx types.Tx) {
	go func() { r.txChan <- tx }()
}

func (r *Reactor) broadcastTxRoutine(peer p2p.Peer) {
	for {
		if !peer.IsRunning() {
			return
		}

		select {
		case tx := <-r.txChan:
			seq := uint64(0)
			vote := protowendy.NewVote(seq, tx.Hash())
			bz := protowendy.MustMarshal(vote)
			peer.Send(WendyChannel, bz)
			log.Printf("Sent vote: %s", vote)
		}
	}
}

func (r *Reactor) AddPeer(peer p2p.Peer) {
	go r.broadcastTxRoutine(peer)
}

func (r *Reactor) Receive(chID byte, peer p2p.Peer, msgBytes []byte) {
	vote := &protowendy.Vote{}
	protowendy.MustUnmarshal(msgBytes, vote)
	log.Printf("got vote: %s", vote)
}

func (r *Reactor) InitPeer(peer p2p.Peer) p2p.Peer {
	return peer
}

func (r *Reactor) GetChannels() []*conn.ChannelDescriptor {
	return []*conn.ChannelDescriptor{
		{ID: WendyChannel, Priority: 5},
	}
}

func (r *Reactor) RemovePeer(peer p2p.Peer, reason interface{}) {}
