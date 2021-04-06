package mempool

import (
	cfg "github.com/tendermint/tendermint/config"
	"github.com/tendermint/tendermint/p2p"
)

const (
	MempoolChannel = byte(0x30)
)

type Reactor struct {
	p2p.BaseReactor
}

func NewReactor(config *cfg.MempoolConfig, mempool *Mempool) *Reactor {
	memR := &Reactor{}
	memR.BaseReactor = *p2p.NewBaseReactor("Mempool", memR)
	return memR
}
