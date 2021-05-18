<img width="654" alt="w" src="https://user-images.githubusercontent.com/13255539/94933906-ee1d5b80-04c2-11eb-96f1-f65cde7ce83f.png">

_The good little fairness widget_

[![Build](https://github.com/vegaprotocol/wendy/actions/workflows/test.yml/badge.svg)](https://github.com/vegaprotocol/wendy/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/vegaprotocol/wendy.svg)](https://pkg.go.dev/github.com/vegaprotocol/wendy)
[![Tag](https://badgen.net/github/tag/vegaprotocol/wendy)](https://github.com/vegaprotocol/wendy/releases/latest)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/vegaprotocol/wendy/blob/main/LICENSE.md)

Wendy acts as an additional widget for an existing blockchain, and is largely agnostic to the underlying blockchain and its security assumptions.
Furthermore, it is possible to apply it to the protocol only for a subset of the transactions, and thus run several independent fair markets on the same chain.
We have implemented Wendy to run on a simulator to get first performance estimates.
As Wendy runs parallel to the actual blockchain, the core impact it has (apart from adding some network traffic) is that some transactions are put into a later block than they would be without fairness.

This repository contains a simulation of [Wendy](https://eprint.iacr.org/2020/885), a protocol for implementing different concepts of fairness.
This implementation is not close to production ready, but it does demonstrate how Wendy impacts performance given various parameters.

# Research paper
Dr Klaus Kursawe's original research paper is available on [IACR](https://eprint.iacr.org/2020/885) or on the [Vega website](https://vega.xyz/background#published-papers). Or you can watch him talk through the paper on [YouTube](https://www.youtube.com/watch?v=tU3CYpT5-qM):

[![Wendy, the good little fairness widget](https://img.youtube.com/vi/tU3CYpT5-qM/0.jpg)](https://www.youtube.com/watch?v=tU3CYpT5-qM)

# Tendermint
We are currently working on a Wendy implementation for [Tendermint](https://github.com/vegaprotocol/wendy/blob/main/tendermint/README.md).
Wendy is implemented as a mempool replacement.

# Notes
The initial Wendy implementation can be found under [v0.0.1](https://github.com/vegaprotocol/wendy/tree/v0.0.1) tag.
