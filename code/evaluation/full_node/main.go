// Author: xxxxx.
package main

import (
	"crypto/rand"
	"encoding/gob"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"

	sig "example.com/light-client-signature"
	"github.com/filecoin-project/go-jsonrpc"
)

type testProver struct {
	Files      int
	Quorum     int
	GenesisApk sig.PublicKey
	Breaks     []big.Int
	Apks       []sig.PublicKey
	Signatures [][][]byte
	Multisigs  [][]byte
}

func newTestProver(committee, quorum, length, periodLength int, directory string) (*testProver, error) {
	t := time.Now()
	// Create fake committee chain of `length` epochs.
	// Within each period of length `periodLength`, committee changes do not
	// touch a static core quorum (the first `quorum` members).
	// Only after every `periodLength`th epoch, the core quorum changes.
	genesisKeys := make([]*sig.PrivateKey, quorum)
	var genesisApk sig.PublicKey
	for i := 0; i < quorum; i++ {
		privKey, _ := sig.GenerateKey(rand.Reader)
		genesisKeys[i] = privKey
		genesisApk.A.Add(&genesisApk.A, &privKey.PublicKey.A)
	}
	nBreaks := int(math.Ceil(float64(length) / float64(periodLength)))
	apks := make([]sig.PublicKey, nBreaks)
	breaks := make([]big.Int, nBreaks)
	sigs := make([][][]byte, length)
	multiSigs := make([][]byte, length)

	// Parallelize
	n := 1 // runtime.GOMAXPROCS(0) // TODO: n>1 sometimes leads to verification failure in TestUpdate* (concurrency bug?!)
	if n > length {
		n = length
	}
	done := make(chan int)
	batchDone := make(chan int)
	keyChans := make([]chan []*sig.PrivateKey, n)
	perCPU := length / n
	leftover := length - n*perCPU
	var start, end, offset int
	for k := 0; k < n; k++ {
		keyChans[k] = make(chan []*sig.PrivateKey, 1)
		start = k*perCPU + offset
		end = start + perCPU
		if leftover > 0 {
			end++
			offset++
			leftover--
		}
		go func(a, b, kk int) {
			// First, create new committees at break points in my interval [a,b)
			firstBreak := int(math.Ceil(float64(a)/float64(periodLength))) * periodLength
			myBreaks := int(math.Ceil(float64(b-firstBreak) / float64(periodLength)))
			privKeys := make([][]*sig.PrivateKey, myBreaks)
			for i := firstBreak; i < b; i = i + periodLength {
				idx := (i - firstBreak) / periodLength
				privKeys[idx] = make([]*sig.PrivateKey, quorum)
				for j := 0; j < quorum; j++ {
					privKey, _ := sig.GenerateKey(rand.Reader)
					privKeys[idx][j] = privKey
					apks[i/periodLength].A.Add(&apks[i/periodLength].A, &privKey.PublicKey.A)
				}
				breaks[i/periodLength] = *big.NewInt(int64(i + 1))
				if i+periodLength >= b {
					toSend := make([]*sig.PrivateKey, quorum)
					copy(toSend, privKeys[idx])
					keyChans[kk] <- toSend
				}
			}
			signingKeys := make([]*sig.PrivateKey, quorum)
			if a == 0 {
				signingKeys = genesisKeys
			} else {
				signingKeys = <-keyChans[kk-1]
			}
			if firstBreak >= b {
				keyChans[kk] <- signingKeys
			}

			// The static quora sign the apks
			for i := a; i < b; i++ {
				quorumSigs := make([][]byte, quorum)
				breakPoint := (i%periodLength == 0)
				if !breakPoint {
					for j := 0; j < quorum; j++ {
						quorumSigs[j], _ = signingKeys[j].SignNoBreak(big.NewInt(int64(i + 1)))
					}
				} else {
					for j := 0; j < quorum; j++ {
						quorumSigs[j], _ = signingKeys[j].SignBreak(big.NewInt(int64(i+1)), apks[i/periodLength])
						signingKeys[j] = privKeys[(i-firstBreak)/periodLength][j]
					}
				}
				sigs[i] = quorumSigs
				multiSigs[i] = sig.AggregateSignatures(quorumSigs)

				if (i-a)%(1<<10) == (1<<10 - 1) {
					batchDone <- (1 << 10)
				}
			}
			done <- 1
		}(start, end, k)
	}

	sofar := 0
	for finished := 0; finished < n; {
		select {
		case batch := <-batchDone:
			sofar = sofar + batch
			log.Printf("creating chain . . . %v/%v\n", sofar, length)
		case <-done:
			finished++
		}
	}
	log.Printf("created chain for committee size %v and length %v in %v\n", committee, length, time.Since(t))

	p := &testProver{
		Quorum:     quorum,
		GenesisApk: genesisApk,
		Breaks:     breaks,
		Apks:       apks,
		Signatures: sigs,
		Multisigs:  multiSigs,
	}
	err := p.save(directory)
	if err != nil {
		return &testProver{}, err
	}
	log.Println("saved chain and initial committee (nextk0.gob)")
	return p, nil
}

func (p *testProver) sizeWithoutSigs() int {
	size := 8                     // Quorum
	size = size + 48              // GenesisApk
	size = size + len(p.Breaks)*8 // Breaks
	size = size + len(p.Apks)*48  // Apks
	return size
}

func (p *testProver) sizeSigs() int {
	size := len(p.Signatures) * p.Quorum * 96 // Signatures
	size = size + len(p.Multisigs)*96         // Multisigs
	return size
}

func (p *testProver) save(directory string) error {
	// Split across several files of max size 2 GiB.
	// The first file includes the number of files and all data except for the signatures.
	nFiles := int(math.Ceil(float64(p.sizeSigs()) / float64(1<<31)))
	f1, err := os.Create(filepath.Join(directory, "committee_chain_0.gob"))
	if err != nil {
		return err
	}
	defer f1.Close()
	enc := gob.NewEncoder(f1)
	err = enc.Encode(testProver{
		Files:      nFiles + 1,
		Quorum:     p.Quorum,
		GenesisApk: p.GenesisApk,
		Apks:       p.Apks,
		Breaks:     p.Breaks,
	})
	if err != nil {
		return err
	}

	perFile := int(math.Ceil(float64(len(p.Signatures)) / float64(nFiles)))
	var pr testProver
	for i := 0; i < nFiles; i++ {
		if (i+1)*perFile >= len(p.Signatures) {
			pr.Signatures = p.Signatures[i*perFile:]
			pr.Multisigs = p.Multisigs[i*perFile:]
		} else {
			pr.Signatures = p.Signatures[i*perFile : (i+1)*perFile]
			pr.Multisigs = p.Multisigs[i*perFile : (i+1)*perFile]
		}
		f, err := os.Create(filepath.Join(directory, fmt.Sprintf("committee_chain_%v.gob", i+1)))
		if err != nil {
			return err
		}
		defer f.Close()
		enc := gob.NewEncoder(f)
		err = enc.Encode(pr)
		if err != nil {
			return err
		}
	}

	f2, err := os.Create(filepath.Join(directory, "nextk0.gob"))
	if err != nil {
		return err
	}
	defer f2.Close()
	enc = gob.NewEncoder(f2)
	err = enc.Encode(p.GenesisApk)
	if err != nil {
		return err
	}
	return nil
}

func newTestProverFromFile(directory string) (*testProver, error) {
	f, err := os.Open(filepath.Join(directory, "committee_chain_0.gob"))
	if err != nil {
		return &testProver{}, err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	var prover testProver
	err = dec.Decode(&prover)
	if err != nil {
		return &testProver{}, err
	}

	for i := 1; i < prover.Files; i++ {
		f, err := os.Open(filepath.Join(directory, fmt.Sprintf("committee_chain_%v.gob", i)))
		if err != nil {
			return &testProver{}, err
		}
		defer f.Close()
		dec := gob.NewDecoder(f)
		var pr testProver
		err = dec.Decode(&pr)
		if err != nil {
			return &testProver{}, err
		}
		prover.Signatures = append(prover.Signatures, pr.Signatures...)
		prover.Multisigs = append(prover.Multisigs, pr.Multisigs...)
	}
	log.Printf("loaded chain of length %v from %v files\n", len(prover.Signatures), prover.Files)
	return &prover, nil
}

// UpdateRedundant aggregates all signatures from scratch each time
func (p *testProver) UpdateRedundant(start, end *big.Int) (sig.LightClientProof, error) {
	return p.update(false, start, end)
}

// UpdateMulti uses pre-aggregated multisignatures
func (p *testProver) UpdateMulti(start, end *big.Int) (sig.LightClientProof, error) {
	return p.update(true, start, end)
}

func (p *testProver) update(withPrecomp bool, start, end *big.Int) (sig.LightClientProof, error) {
	if end.Int64() > int64(len(p.Signatures)) {
		end.SetInt64(int64(len(p.Signatures)))
	}
	var first int
	last := len(p.Breaks)
	var foundFirst bool
	for i, b := range p.Breaks {
		if !foundFirst && (b.Cmp(start) >= 0) {
			first = i
			foundFirst = true
		}
		if b.Cmp(end) > 0 {
			last = i
			break
		}
	}

	breaks := make([]big.Int, last-first)
	copy(breaks, p.Breaks[first:last])
	breaks = append(breaks, *end)

	apks := make([]sig.PublicKey, last-first)
	copy(apks, p.Apks[first:last])

	var toAggregate [][]byte
	for j := start.Int64(); j <= end.Int64(); j++ {
		if withPrecomp {
			toAggregate = append(toAggregate, p.Multisigs[j-1])
		} else {
			toAggregate = append(toAggregate, p.Signatures[j-1]...)
		}
	}

	s := sig.AggregateSignatures(toAggregate)

	return sig.LightClientProof{
		NextApks:  apks,
		Signature: s,
		Breaks:    breaks,
	}, nil
}

// ------------------------------

// RpcHandler contains the RPCs that we serve
type RpcHandler interface {
	Dummy(int) []byte
	UpdateRedundant(*big.Int) (sig.LightClientProof, error)
	UpdateMulti(*big.Int) (sig.LightClientProof, error)
}

type limitedProver struct {
	p     *testProver
	limit *big.Int
}

func (lp *limitedProver) Dummy(howmany int) []byte {
	b := make([]byte, howmany)
	return b
}

func (lp *limitedProver) UpdateRedundant(epoch *big.Int) (sig.LightClientProof, error) {
	t := time.Now()
	result, err := lp.p.UpdateRedundant(epoch, lp.limit)
	log.Printf("[chain length %v] served UpdateRedundant in %v\n", lp.limit, time.Since(t))
	return result, err
}

func (lp *limitedProver) UpdateMulti(epoch *big.Int) (sig.LightClientProof, error) {
	t := time.Now()
	result, err := lp.p.UpdateMulti(epoch, lp.limit)
	log.Printf("[chain length %v] served UpdateMulti in %v\n", lp.limit, time.Since(t))
	return result, err
}

// startRPC starts goroutines to listen for RPCs and returns immediately.
// There are different RPC namespaces for different chain lengths.
func (p *testProver) startRPC(addr string) {
	rpcServer := jsonrpc.NewServer()
	max := int(math.Log2(float64(len(p.Signatures))))
	for i := 0; i <= max; i++ {
		prover := &limitedProver{p, big.NewInt(int64(1 << i))}
		rpcServer.Register(fmt.Sprintf("Prover%v", i), RpcHandler(prover))
	}
	go func() {
		s := &http.Server{
			Addr:    addr,
			Handler: rpcServer,
		}
		log.Printf("serving RPCs on http://%v\n", s.Addr)
		err := s.ListenAndServe()
		if err != nil {
			log.Fatalln("ListenAndServe:", err)
		}
	}()
}

func main() {
	var prover *testProver
	directory := "."
	logfile, err := os.OpenFile(filepath.Join(directory, "ours_128_wan_fullnode.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer logfile.Close()
	log.SetOutput(logfile)
	files, err := os.ReadDir(directory)
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		if file.Name() == "committee_chain_0.gob" {
			prover, err = newTestProverFromFile(directory)
			if err != nil {
				panic(err)
			}
			break
		}
	}
	if prover == nil {
		committee := 128
		quorum := 86
		length := 1 << 10
		periodLength := 2000
		prover, err = newTestProver(committee, quorum, length, periodLength, directory)
		if err != nil {
			panic(err)
		}
	}
	prover.startRPC(":7890")
	select {}
}
