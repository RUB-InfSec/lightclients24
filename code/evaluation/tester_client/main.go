// Author: xxxxx.
//
// # Benchmark light client updates as served by the testProver
//
// Usage: binary hostname port log_max_len n_seq
// - hostname and port specify a testProver JSON-RPC endpoint
// - log_max_len specifies the RPC namespaces Prover1, ..., Prover{log_max_len}
// - n_seq is the number of requests
package main

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"

	sig "example.com/light-client-signature"
	"github.com/filecoin-project/go-jsonrpc"
)

func sequentialRequests(prover sig.Prover, genesis sig.PublicKey, nSeq int) error {
	for i := 0; i < nSeq; i++ {
		lc := sig.NewLightClient(genesis)
		tTotal, tLocal, err := lc.UpdateTiming(prover)
		if err != nil {
			return err
		}
		log.Printf("updated light client in %v; local computation: %v\n", tTotal, tLocal)
	}
	return nil
}

// proverEndpoint can be used to fulfil the sig.Prover interface with any function
type proverEndpoint func(*big.Int) (sig.LightClientProof, error)

func (f proverEndpoint) Update(i *big.Int) (sig.LightClientProof, error) {
	return f(i)
}

func main() {
	// set up logging
	directory := "."
	logfile, err := os.OpenFile(filepath.Join(directory, "results.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer logfile.Close()
	log.SetOutput(logfile)

	// parse arguments
	address := "http://" + os.Args[1] + ":" + os.Args[2]
	logMaxLen, err := strconv.Atoi(os.Args[3])
	if err != nil {
		panic(err)
	}
	nSeq, err := strconv.Atoi(os.Args[4])
	if err != nil {
		panic(err)
	}
	log.Printf("starting tester client for the testProver at %v; n_seq=%v\n", address, nSeq)

	// read genesis committee and quorum
	f, err := os.Open(filepath.Join(directory, "nextk0.gob"))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var genesis sig.PublicKey
	err = dec.Decode(&genesis)
	if err != nil {
		panic(err)
	}
	log.Printf("took quorum apk from nextk0.gob\n")

	// set up RPCs
	var _fullNode struct {
		Dummy           func(int) string
		UpdateRedundant func(*big.Int) (sig.LightClientProof, error)
		UpdateMulti     func(*big.Int) (sig.LightClientProof, error)
	}
	for j := 0; j <= logMaxLen; j++ {
		closer, err := jsonrpc.NewClient(context.Background(), address, fmt.Sprintf("Prover%v", j), &_fullNode, nil)
		if err != nil {
			panic(err)
		}
		defer closer()
		fullNodeMulti := proverEndpoint(func(epoch *big.Int) (sig.LightClientProof, error) {
			// perform RPC
			return _fullNode.UpdateMulti(epoch)
		})
		fullNodeRedundant := proverEndpoint(func(epoch *big.Int) (sig.LightClientProof, error) {
			// perform RPC
			return _fullNode.UpdateRedundant(epoch)
		})

		// request lots of light client updates
		if j == 0 {
			log.Println("starting network latency baseline estimate")
			for k := 0; k < 20; k++ {
				for i := 0; i < 5; i++ {
					t := time.Now()
					_fullNode.Dummy(1 << k)
					log.Printf("dummy request for %v bytes took %v\n", 1<<k, time.Since(t))
				}
			}
		}
		log.Printf("starting benchmarks for Prover%v.UpdateRedundant\n", j)
		err = sequentialRequests(fullNodeRedundant, genesis, nSeq)
		if err != nil {
			panic(err)
		}
		log.Printf("starting benchmarks for Prover%v.UpdateMulti\n", j)
		err = sequentialRequests(fullNodeMulti, genesis, nSeq)
		if err != nil {
			panic(err)
		}
	}
	fmt.Println("done")
}
