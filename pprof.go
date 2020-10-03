package main

import (
	"fmt"
	"os"
	"path/filepath"
	rt "runtime"
	"runtime/pprof"
	"time"
)

type pprofh struct {
	cpuprofile, memprofile string
}

func newPPROF(outdir string) (*pprofh, error) {
	// ensure the pprof output directory esist
	if err := ensureDir(outdir); err != nil {
		return nil, err
	}

	// compute the output files names
	t := time.Now()
	p := pprofh{
		cpuprofile: filepath.Join(outdir, fmt.Sprintf("cpu-%s.pprof", t.Format("2006-01-02-15-04-05"))),
		memprofile: filepath.Join(outdir, fmt.Sprintf("mem-%s.pprof", t.Format("2006-01-02-15-04-05"))),
	}

	// create the file to be sent to the CPU profiler
	f, err := os.Create(p.cpuprofile)
	if err != nil {
		return nil, err
	}
	// start the profiling
	if err := pprof.StartCPUProfile(f); err != nil {
		return nil, err
	}

	return &p, nil
}

func (p *pprofh) Stop() error {
	// wait for mem profile to be written before stopping the CPU profile
	defer pprof.StopCPUProfile()

	f, err := os.Create(p.memprofile)
	if err != nil {
		return fmt.Errorf("unable to create memprofile file: %w", err)
	}

	rt.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("unable to write memprofile file: %w", err)
	}
	f.Close()
	return nil
}

func ensureDir(outdir string) error {
	oi, err := os.Stat(outdir)
	if !os.IsNotExist(err) {
		return err
	}
	if err == nil && !oi.IsDir() {
		return fmt.Errorf("%v is not a directory", outdir)
	}
	return os.MkdirAll(outdir, os.ModePerm)

}
