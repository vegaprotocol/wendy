package main

import (
	"flag"
	"fmt"
)

var args struct {
	r               int
	n               int
	t               int
	debugLeader     int
	delay           int
	msgDelay        int
	msgRnd          int
	blockchainDelay int
	blockchainRnd   int
	runtime         int
}

func init() {
	flag.IntVar(&args.r, "r", 10, "debug switch (ignore for now)")
	flag.IntVar(&args.n, "n", 5, "number of validators (max 19)")
	flag.IntVar(&args.t, "t", 1, "number of allowed traitors")
	flag.IntVar(&args.debugLeader, "debugLeader", 11, "which party do we want to watch, if > n, none")
	flag.IntVar(&args.delay, "delay", 1, "speed at which new transactions are inserted")
	flag.IntVar(&args.msgDelay, "msgDelay", 100, "message basic transmission time")
	flag.IntVar(&args.msgRnd, "msgRnd", 200, "random interval to be added to msgDelay")
	flag.IntVar(&args.blockchainDelay, "blockchainDelay", 100, "block processing time")
	flag.IntVar(&args.blockchainRnd, "blockchainRnd", 0, "random part thereof")
	flag.IntVar(&args.runtime, "runtime", 0, "number of ticks before the simulation stops, run forever by default")
}

func main() {
	flag.Parse()

	fmt.Println("Wendy starting")
	initWendy()
	r = args.r
	n = args.n
	t = args.t
	debugLeader = args.debugLeader
	delay = args.delay
	msgDelay = args.msgDelay
	msgRnd = args.msgRnd
	blockchainDelay = args.blockchainDelay
	blockchainRnd = args.blockchainRnd
	runtime = args.runtime
	//messagebuffer={};
	sendMessageWithTime("NIL", "BlockTrigger", 0, 0, worldTime+101)
	networkNew()
}
