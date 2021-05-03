Tendermint for Wendy (work in progress)
--

This pkg contains the code required to integrate Wendy and Tendermint.
Since Tendermint's API does not support injecting a [User defined mempool](https://github.com/tendermint/tendermint/issues/62430) we re-implemented the `./node/` package where our custom mempool is injected, the mempool lives in the `./mempool` directory.
