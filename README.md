# Wendy, the good little fairness widget

An simulation of [Wendy](https://eprint.iacr.org/2020/885), a protocol for implementing different concepts of fairness. 

Wendy acts as an aditional widget for an existing blockchain, and is largely agnostic to the underlying blockchain and its security assumptions. Furthermore, it is possible to apply a the protocol only for a subset of the transactions, and thus run several independent fair markets on the same chain. We have implemented Wendy to run on a simulator to get first performance estimates. As Wendy runs parallel to the actual blockchain, the core impact it has (apart from adding some network traffic) is that some transactions are put into a later block than they would be without fairness.

## Results
- Transactions that relate to a market that needs no fairness are unimpacted; there is no measurable delay here.
- The primary factor under good circumstance is the ratio of blockchain speed to message delivery time. Under optimal conditions, the number of transactions pushed to the next block is exactly that ratio. This means that if Wendy is used as a pre-protocol for:
  - [Ethereum](https://github.com/ethereum/go-ethereum), __there is practically no performance impact__
  - a higher speed protocol like [Hotstuff](https://github.com/hot-stuff/libhotstuff) or [Tendermint](https://github.com/tendermint/tendermint), that ratio can go up . Results suggest __10-15% of transactions would be pushed to the next block__, depending on network conditions).
- In a more erratic network the number of transactions that where delayed in out simulation slightly more than doubled when compared to the best case.

## Running the Wendy simulator
Clone this repository
```bash
go build
./wendy
```
If you want more detail, you can see more output by setting ```-debugLeader``` to be one of the validators (default: 5). To tweak the parameters of the simulation, run `wendy -h` to see the available options. You can [see some sample output below](#sample-output).

# Research paper
Dr Klaus Kursawe's original research paper by Kis available on [IACR](https://eprint.iacr.org/2020/885) or on the [Vega website](https://vega.xyz/background#published-papers). Or you can watch him talk through the paper on [YouTube](https://www.youtube.com/watch?v=tU3CYpT5-qM):

[![Wendy, the good little fairness widget](https://img.youtube.com/vi/tU3CYpT5-qM/0.jpg)](https://www.youtube.com/watch?v=tU3CYpT5-qM)


# Sample output
```
Wendy starting
Wendy initializing
Proposing new block:  0 null
Number of TXs in block:  0
Number of Txs delayed by Wendy:  0
Out of order votes in leader buffer:  20
Worldtime is : 101
Seq number of  1 is  -1
Seq number of  2 is  -1
Seq number of  3 is  -1
Seq number of  4 is  -1
Seq number of  5 is  -1
We've finished a block with  0 entries
-----------------------------------------------
[]
Proposing new block:  0 null
Number of TXs in block:  0
Number of Txs delayed by Wendy:  26
Out of order votes in leader buffer:  20
Worldtime is : 202
Seq number of  1 is  25
Seq number of  2 is  -1
Seq number of  3 is  -1
Seq number of  4 is  -1
Seq number of  5 is  -1
We've finished a block with  0 entries
-----------------------------------------------
[]
Proposing new block:  2 ["701","1101"]
Number of TXs in block:  2
Number of Txs delayed by Wendy:  103
Out of order votes in leader buffer:  20
Worldtime is : 303
Seq number of  1 is  105
Seq number of  2 is  0
Seq number of  3 is  1
Seq number of  4 is  -1
Seq number of  5 is  -1
We've finished a block with  2 entries
```
