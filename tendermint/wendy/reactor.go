package wendy

import (
	"sync/atomic"

	"github.com/tendermint/tendermint/libs/clist"
	"github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/p2p/conn"
	"github.com/tendermint/tendermint/types"

	protowendy "github.com/vegaprotocol/wendy/proto/wendy"
)

const (
	WendyChannel = byte(0x99)
)

type Reactor struct {
	p2p.BaseReactor

	id     string
	logger log.Logger

	txChan chan (types.Tx)
	seq    uint64
	votes  *clist.CList
}

func NewReactor(id p2p.ID) *Reactor {
	r := &Reactor{
		id:     string(id),
		logger: log.NewNopLogger(),
		txChan: make(chan types.Tx),
	}
	r.BaseReactor = *p2p.NewBaseReactor("Wendy", r)
	return r
}

func (r *Reactor) WithLogger(logger log.Logger) *Reactor {
	r.logger = logger.With("module", "wendy")
	return r
}

// OnNewTx is a handler for a new incoming Tx.
// Since it is designed to react upon new Tx on the mempool, the signature
// satisfies the mempool.NotifyFunc.
func (r *Reactor) OnNewTx(tx types.Tx) {
	go func() { r.txChan <- tx }()
}

// nextSeq returns the next sequence number.
// This function is safe for concurrent access.
func (r *Reactor) nextSeq() uint64 {
	// atomic.AddUint64 returns the new value
	return atomic.AddUint64(&r.seq, 1)
}

func (r *Reactor) logVote(msg string, vote *protowendy.Vote, peer p2p.Peer) {
	r.logger.Debug(msg, "sender", vote.Sender[0:4], "peer", peer.ID()[0:4], "seq", vote.Sequence, "hash", vote.TxHash)
}

// newVote returns a new vote given a tx. It will generate other fields based
// on the internal state.
func (r *Reactor) newVote(tx types.Tx) *protowendy.Vote {
	return protowendy.NewVote(r.id, r.nextSeq(), tx.Hash())
}

func (r *Reactor) broadcastTxRoutine(peer p2p.Peer) {
	for {
		if !peer.IsRunning() {
			return
		}

		select {
		case tx := <-r.txChan:
			vote := r.newVote(tx)
			bz := protowendy.MustMarshal(vote)
			peer.Send(WendyChannel, bz)
			r.logVote("Vote sent", vote, peer)
		}
	}
}

func (r *Reactor) AddPeer(peer p2p.Peer) {
	go r.broadcastTxRoutine(peer)
}

func (r *Reactor) Receive(chID byte, peer p2p.Peer, msgBytes []byte) {
	vote := &protowendy.Vote{}
	protowendy.MustUnmarshal(msgBytes, vote)
	r.logVote("Vote received", vote, peer)
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
