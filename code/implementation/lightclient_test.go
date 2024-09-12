// Based on a BLS signature implementation from gnark-crypto.
// Author (modifications): xxxxx.
// Original copyright notice:
//
// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package lightclientsignature

import (
	"crypto/rand"
	"math/big"
	"testing"
)

type testProver struct {
	quorum        int
	genesisApk    PublicKey
	committees    [][]PublicKey
	breaks        []big.Int
	apks          []PublicKey
	signatures    [][][]byte
	aggSignatures [][]byte
}

func newTestProver(committee, quorum, length, periodLength int) *testProver {
	// Create fake committee chain of `length` epochs.
	// Within each period of length `periodLength`, committee changes do not
	// touch a static core quorum (the first `quorum` members).
	// Only after every `periodLength`th epoch, the core quorum changes.
	genesis := make([]PublicKey, committee)
	var genesisApk PublicKey
	signingKeys := make([]*PrivateKey, quorum)
	for i := 0; i < committee; i++ {
		privKey, _ := GenerateKey(rand.Reader)
		genesis[i] = privKey.PublicKey
		if i < quorum {
			signingKeys[i] = privKey
			genesisApk.A.Add(&genesisApk.A, &privKey.PublicKey.A)
		}
	}

	committees := make([][]PublicKey, length)
	apks := make([]PublicKey, length/periodLength)
	breaks := make([]big.Int, length/periodLength)
	sigs := make([][][]byte, length)
	aggSigs := make([][]byte, length)
	prev := genesis
	for i := 0; i < length; i++ {
		newCommittee := make([]PublicKey, length)
		newPrivKeys := make([]*PrivateKey, quorum)
		var newApk PublicKey
		quorumSigs := make([][]byte, quorum)
		breakPoint := (i%periodLength == 0)
		if !breakPoint {
			// Change all committee members except for a core quorum
			copy(newCommittee[:quorum], prev[:quorum])
			for j := quorum; j < committee; j++ {
				privKey, _ := GenerateKey(rand.Reader)
				newCommittee[j] = privKey.PublicKey
			}

			// The core quorum signs
			for j := 0; j < quorum; j++ {
				s, _ := signingKeys[j].SignNoBreak(big.NewInt(int64(i + 1)))
				quorumSigs[j] = s
			}
		} else {
			// Change all committee members including the core quorum
			for j := 0; j < committee; j++ {
				privKey, _ := GenerateKey(rand.Reader)
				newCommittee[j] = privKey.PublicKey
				if j < quorum {
					newPrivKeys[j] = privKey
					newApk.A.Add(&newApk.A, &privKey.PublicKey.A)
				}
			}

			// The core quorum signs and gets replaced
			for j := 0; j < quorum; j++ {
				s, _ := signingKeys[j].SignBreak(big.NewInt(int64(i+1)), newApk)
				quorumSigs[j] = s
				signingKeys[j] = newPrivKeys[j]
			}

			breaks = append(breaks, *big.NewInt(int64(i + 1)))
			apks = append(apks, newApk)
		}
		committees[i] = newCommittee
		prev = newCommittee
		sigs[i] = quorumSigs
		aggSigs[i] = AggregateSignatures(quorumSigs)
	}

	return &testProver{
		quorum:        quorum,
		genesisApk:    genesisApk,
		committees:    committees,
		breaks:        breaks,
		apks:          apks,
		signatures:    sigs,
		aggSignatures: aggSigs,
	}
}

func (p *testProver) Update(epoch *big.Int) (LightClientProof, error) {
	breaks := []big.Int{}
	newKeys := []PublicKey{}
	for i, b := range p.breaks {
		if b.Cmp(epoch) >= 0 {
			breaks = p.breaks[i:]
			newKeys = p.apks[i:]
			break
		}
	}
	current := big.NewInt(int64(len(p.committees)))
	if breaks[len(breaks)-1].Cmp(current) != 0 {
		breaks = append(breaks, *current)
	}

	toAggregate := [][]byte{}
	var prev big.Int
	prev.Set(epoch)
	for _, b := range breaks {
		// collect the core quorum's signatures from prev to b
		for j := int(prev.Int64()); j <= int(b.Int64()); j++ {
			toAggregate = append(toAggregate, p.signatures[j-1]...)
		}
		prev.Add(&b, one)
	}

	return LightClientProof{
		NextApks:  newKeys,
		Signature: AggregateSignatures(toAggregate),
		Breaks:    breaks,
	}, nil
}

func TestUpdate(t *testing.T) {
	committee := 10
	quorum := 7
	length := 20
	periodLength := length / 5
	prover := newTestProver(committee, quorum, length, periodLength)
	genesisApk := prover.genesisApk

	lc := NewLightClient(genesisApk)

	err := lc.Update(prover)
	if err != nil {
		t.Fatal(err)
	}
	if lc.Epoch.Cmp(big.NewInt(int64(length+1))) != 0 {
		t.Fatalf("wrong final epoch: want %v; got %v", length+1, lc.Epoch.Int64())
	}
	if !lc.QuorumApk.A.Equal(&prover.apks[len(prover.apks)-1].A) {
		t.Fatal("wrong final apk")
	}
}
