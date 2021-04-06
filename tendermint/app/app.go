package app

import (
	"fmt"

	"github.com/tendermint/tendermint/mempool"

	abci "github.com/tendermint/tendermint/abci/types"
)

type App struct {
	abci.BaseApplication
	mempool mempool.Mempool
}

func New() *App {
	return &App{}
}

func (app *App) SetMempool(mp mempool.Mempool) {
	app.mempool = mp
}

func (ap *App) CheckTx(req abci.RequestCheckTx) abci.ResponseCheckTx {
	fmt.Printf("CheckTx(%8s): (%s)\n", req.Type, string(req.Tx))
	return abci.ResponseCheckTx{Code: abci.CodeTypeOK}
}
