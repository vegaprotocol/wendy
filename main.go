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
	init_wendy()
	r = args.r
	n = args.n
	t = args.t
	debug_leader = args.debugLeader
	delay = args.delay
	msg_delay = args.msgDelay
	msg_rnd = args.msgRnd
	blockchain_delay = args.blockchainDelay
	blockchain_rnd = args.blockchainRnd
	runtime = args.runtime
	//messagebuffer={};
	send_message_withtime("NIL", "BlockTrigger", 0, 0, worldtime+101)
	network_new()
}
