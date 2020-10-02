<img width="654" alt="w" src="https://user-images.githubusercontent.com/13255539/94933906-ee1d5b80-04c2-11eb-96f1-f65cde7ce83f.png">

The good little fairness widget

Wendy acts as an aditional widget for an existing blockchain, and is largely agnostic to the underlying blockchain and its security assumptions. Furthermore, it is possible to apply a the protocol only for a subset of the transactions, and thus run several independent fair markets on the same chain. We have implemented Wendy to run on a simulator to get first performance estimates. As Wendy runs parallel to the actual blockchain, the core impact it has (apart from adding some network traffic) is that some transactions are put into a later block than they would be without fairness.

This repository contains a simulation of [Wendy](https://eprint.iacr.org/2020/885), a protocol for implementing different concepts of fairness. It's not even close to production ready, but it does demonstrate how Wendy impacts performance given various parameters.

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
Proposing new block:  23 ["1801","901","3601","4501","2701","10801","9901","7201","9001","11701","8101","5401","6301","12601","13501","15301","14401","16201","18901","20701","17101","21601","18001"]
Number of TXs in block:  23
Number of Txs delayed by Wendy:  22
Out of order votes in leader buffer:  20
Worldtime is : 602
Seq number of  1 is  44
Seq number of  2 is  15
Seq number of  3 is  16
Seq number of  4 is  12
Seq number of  5 is  19
We've finished a block with  23 entries
-----------------------------------------------
Msg 22:922     Msg 21:886     Msg 20:931     Msg 19:895     Msg 18:913     Msg 17:940     Msg 16:958     Msg 15:949     Msg 14:967     Msg 13:976     Msg 12:1039     Msg 11:1048     Msg 10:1021     Msg 9:985     Msg 8:1012     Msg 7:1030     Msg 6:1003     Msg 5:994     Msg 4:1075     Msg 3:1057     Msg 2:1066     Msg 1:1093     Msg 0:1084     -------------------------------------
Average time:  994
[1801 901 3601 4501 2701 10801 9901 7201 9001 11701 8101 5401 6301 12601 13501 15301 14401 16201 18901 20701 17101 21601 18001]
Proposing new block:  55 ["22501","23401","27001","30601","24301","19801","27901","28801","25201","26101","29701","33301","31501","32401","34201","36901","36001","37801","42301","40501","41401","38701","35101","45901","39601","43201","45001","52201","51301","50401","44101","46801","47701","49501","48601","54901","54001","58501","60301","56701","55801","57601","59401","53101","62101","61201","63001","64801","69301","67501","66601","68401","65701","63901","70201"]
Number of TXs in block:  55
Number of Txs delayed by Wendy:  22
Out of order votes in leader buffer:  20
Worldtime is : 1103
Seq number of  1 is  99
Seq number of  2 is  68
Seq number of  3 is  72
Seq number of  4 is  71
Seq number of  5 is  70
```

# Footnote
This simulation lets you tweak the parameters and see how much of an impact implementing Wendy would have on the block placement of orders. This is not a full implementation, or a well-tested one. A full implementation of Wendy will be implemented in [Vega](https://vega.xyz).
