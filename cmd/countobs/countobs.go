package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/entrope/gnss/rinex"
)

var njobs = flag.Uint("j", 1, "number of concurrent jobs to launch")

type result struct {
	filename string
	nEpochs  int
	nObs     int
	err      error
}

func reportResults(wg *sync.WaitGroup, results <-chan *result) {
	defer wg.Done()
	for {
		res, ok := <-results
		if !ok {
			return
		}
		if res.err != nil {
			fmt.Printf("%s : %s\n", res.filename, res.err.Error())
		} else {
			fmt.Printf("%s : %d epochs, %d obs\n", res.filename,
				res.nEpochs, res.nObs)
		}
	}
}

func readFiles(wg *sync.WaitGroup, results chan<- *result, filenames <-chan string) {
	defer wg.Done()
	for {
		filename, ok := <-filenames
		if !ok {
			return
		}
		f, err := os.Open(filename)
		if err != nil {
			fmt.Println(err)
			continue
		}
		r := io.Reader(f)
		if strings.HasSuffix(filename, ".gz") {
			if r, err = gzip.NewReader(r); err != nil {
				fmt.Println(filename, ":", err)
			}
		}

		res := &result{filename: filename}
		or := rinex.ObsReader{
			ObsFunc: func(rec rinex.ObservationRecord) error {
				if rec.Year != 0 && rec.Month != 0 && rec.Day != 0 {
					res.nEpochs++
					res.nObs += len(rec.Sat)
				}
				return nil
			},
		}
		res.err = or.Parse(r)
		results <- res
	}
}

func main() {
	flag.Parse()
	if *njobs == 0 {
		*njobs = uint(runtime.NumCPU())
	}

	filenames := make(chan string, 8+*njobs)
	results := make(chan *result, 1+*njobs)
	wg1 := sync.WaitGroup{}
	wg1.Add(1)
	go reportResults(&wg1, results)

	wg2 := sync.WaitGroup{}
	for i := uint(0); i < *njobs; i++ {
		wg2.Add(1)
		go readFiles(&wg2, results, filenames)
	}

	for _, filename := range flag.Args() {
		filenames <- filename
	}

	close(filenames)
	wg2.Wait()
	close(results)
	wg1.Wait()
}
