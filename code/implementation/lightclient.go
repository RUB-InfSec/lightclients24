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
	"math/big"
	"time"
)

type LightClientProof struct {
	NextApks  []PublicKey `json:"next_apks"`
	Signature []byte      `json:"signature"`
	Breaks    []big.Int   `json:"breaks"`
}

// Prover is a full node that returns a LightClientProof when given a start epoch
type Prover interface {
	Update(*big.Int) (LightClientProof, error)
}

// LightClient implements a light client verifier
type LightClient struct {
	Epoch     *big.Int
	QuorumApk PublicKey
	// omitted: state of the blockchain etc. (deployment specific)
}

func NewLightClient(genesisQuorumApk PublicKey) *LightClient {
	return &LightClient{
		Epoch:     big.NewInt(1),
		QuorumApk: genesisQuorumApk,
	}
}

// Update implements the verifier in the light client update protocol.
func (lc *LightClient) Update(prover Prover) error {
	pi, err := prover.Update(lc.Epoch)
	if err != nil {
		return err
	}
	if err := lc.verify(pi); err != nil {
		return err
	}
	lc.QuorumApk = pi.NextApks[len(pi.NextApks)-1]
	lc.Epoch.Add(&pi.Breaks[len(pi.Breaks)-1], one)
	return nil
}

func (lc *LightClient) verify(pi LightClientProof) error {
	// omitted: check correct format and length of proof fields, that break points are increasing, etc.
	apks := append([]PublicKey{lc.QuorumApk}, pi.NextApks...)
	var epoch big.Int
	epoch.Sub(lc.Epoch, one)
	breaks := append([]big.Int{epoch}, pi.Breaks...)
	return VerifyPeriods(apks, pi.Signature, breaks)
}

// UpdateTiming is the same as Update but also returns timings
func (lc *LightClient) UpdateTiming(prover Prover) (time.Duration, time.Duration, error) {
	withProverCall := time.Now()
	pi, err := prover.Update(lc.Epoch)
	if err != nil {
		return time.Duration(0), time.Duration(0), err
	}
	onlyLocal := time.Now()
	if err := lc.verify(pi); err != nil {
		return time.Duration(0), time.Duration(0), err
	}
	lc.QuorumApk = pi.NextApks[len(pi.NextApks)-1]
	lc.Epoch.Add(&pi.Breaks[len(pi.Breaks)-1], one)
	return time.Since(withProverCall), time.Since(onlyLocal), nil
}
