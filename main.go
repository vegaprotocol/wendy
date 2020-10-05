package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
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
	withPPROF       string
}

func init() {
	flag.IntVar(&args.r, "r", 10, "debug switch (ignore for now)")
	flag.IntVar(&args.n, "n", 4, "number of validators (max 19)")
	flag.IntVar(&args.t, "t", 1, "number of allowed traitors")
	flag.IntVar(&args.debugLeader, "debugLeader", 11, "which party do we want to watch, if > n, none")
	flag.IntVar(&args.delay, "delay", 9, "speed at which new transactions are inserted")
	flag.IntVar(&args.msgDelay, "msgDelay", 100, "message basic transmission time")
	flag.IntVar(&args.msgRnd, "msgRnd", 20, "random interval to be added to msgDelay")
	flag.IntVar(&args.blockchainDelay, "blockchainDelay", 1100, "block processing time")
	flag.IntVar(&args.blockchainRnd, "blockchainRnd", 0, "random part thereof")
	flag.IntVar(&args.runtime, "runtime", 0, "number of ticks before the simulation stops, run forever by default")
	flag.StringVar(&args.withPPROF, "withPPROF", "", "a string containing a directory path in which to store the CPU/MEM profile. profiling is not started if empty")
}

func main() {
	flag.Parse()

	if args.n > 19 {
		fmt.Printf("error: n cannot be > 19\n")
		os.Exit(1)
	}

	var (
		pprof *pprofh
		err   error
	)
	if len(args.withPPROF) > 0 {
		// a path is specified, we try start profiling
		pprof, err = newPPROF(args.withPPROF)
		if err != nil {
			fmt.Printf("error: unable to start pprof profiling (%v)", err)
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// we start the main wendy routine
	go func() {
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
		// we are done here, let's cancel the context
		cancel()
	}()

	// catch user signals
	var ch = make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM)
	signal.Notify(ch, syscall.SIGINT)

	// wait for a signal, or wendy to finish
	select {
	case <-ch:
		fmt.Printf("wendy simulation cancelled by user request\n")
	case <-ctx.Done():
		fmt.Printf("wendy simulation done\n")
	}

	// if pprof was enabled we stop it
	if pprof != nil {
		if err := pprof.Stop(); err != nil {
			fmt.Printf("error: could not stop pprof profiling properly (%v)", err)
			os.Exit(1)
		}
	}
}
