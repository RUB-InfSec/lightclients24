// Author: xxxxx.
package main

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"testing"

	sig "example.com/light-client-signature"
	"github.com/filecoin-project/go-jsonrpc"
)

func TestSaveAndLoad(t *testing.T) {
	committee := 10
	quorum := 7
	length := 1 << 12
	periodLength := length / 10
	directory := t.TempDir()
	prover1, err := newTestProver(committee, quorum, length, periodLength, directory)
	if err != nil {
		t.Fatal(err)
	}
	prover2, err := newTestProverFromFile(directory)
	if err != nil {
		t.Fatal(err)
	}
	prover2.Files = 0
	if reflect.DeepEqual(prover1, prover2) == false {
		t.Errorf("got length %v; want length %v", len(prover2.Signatures), len(prover1.Signatures))
	}
}

// proverEndpoint can be used to fulfil the sig.Prover interface with any function
type proverEndpoint func(*big.Int) (sig.LightClientProof, error)

func (f proverEndpoint) Update(i *big.Int) (sig.LightClientProof, error) {
	return f(i)
}

func TestUpdateMultiRPC(t *testing.T) {
	// set up full node
	committee := 10
	quorum := 7
	length := 100
	periodLength := 5
	logMaxLength := int(math.Log2(float64(length)))
	directory := t.TempDir()
	prover, err := newTestProver(committee, quorum, length, periodLength, directory)
	if err != nil {
		t.Fatal(err)
	}
	port := "2345"
	prover.startRPC(":" + port)

	// client side
	var _fullNode struct {
		UpdateMulti func(*big.Int) (sig.LightClientProof, error)
	}
	closer, err := jsonrpc.NewClient(context.Background(), "http://:"+port, fmt.Sprintf("Prover%v", logMaxLength), &_fullNode, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer closer()
	fullNode := proverEndpoint(func(epoch *big.Int) (sig.LightClientProof, error) {
		// perform RPC
		return _fullNode.UpdateMulti(epoch)
	})

	lc := sig.NewLightClient(prover.GenesisApk)
	err = lc.Update(fullNode)
	if err != nil {
		t.Fatal(err)
	}
	want := big.NewInt(int64(1<<logMaxLength + 1))
	if lc.Epoch.Cmp(want) != 0 {
		t.Fatalf("wrong final epoch: got %v, want %v", lc.Epoch, want)
	}
	if lc.QuorumApk.A.Equal(&prover.Apks[1<<logMaxLength/periodLength-1].A) == false {
		t.Fatal("wrong final keys")
	}
}

func TestUpdateRedundantRPC(t *testing.T) {
	// set up full node
	committee := 10
	quorum := 7
	length := 9
	periodLength := 2
	logMaxLength := int(math.Log2(float64(length)))
	directory := t.TempDir()
	prover, err := newTestProver(committee, quorum, length, periodLength, directory)
	if err != nil {
		t.Fatal(err)
	}
	port := "2316"
	prover.startRPC(":" + port)

	// client side
	var _fullNode struct {
		UpdateRedundant func(*big.Int) (sig.LightClientProof, error)
	}
	closer, err := jsonrpc.NewClient(context.Background(), "http://:"+port, fmt.Sprintf("Prover%v", logMaxLength), &_fullNode, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer closer()
	fullNode := proverEndpoint(func(epoch *big.Int) (sig.LightClientProof, error) {
		// perform RPC
		return _fullNode.UpdateRedundant(epoch)
	})

	lc := sig.NewLightClient(prover.GenesisApk)
	err = lc.Update(fullNode)
	if err != nil {
		t.Fatal(err)
	}
	want := big.NewInt(int64(1<<logMaxLength + 1))
	if lc.Epoch.Cmp(want) != 0 {
		t.Fatalf("got %v, want %v", lc.Epoch, want)
	}
	if lc.QuorumApk.A.Equal(&prover.Apks[1<<logMaxLength/periodLength-1].A) == false {
		t.Fatal("wrong final keys")
	}
}
