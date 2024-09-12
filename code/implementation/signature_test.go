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

func TestSignNoBreak(t *testing.T) {
	privKey, _ := GenerateKey(rand.Reader)
	publicKey := privKey.PublicKey

	i := big.NewInt(834)
	sig, err := privKey.SignNoBreak(i)
	if err != nil {
		t.Fatal(err)
	}

	iMinus1 := big.NewInt(1)
	iMinus1.Sub(i, iMinus1)
	err = VerifyPeriods([]PublicKey{publicKey}, sig, []big.Int{*iMinus1, *i})
	if err != nil {
		t.Fatal(err)
	}
}

func TestSignBreak(t *testing.T) {
	privKey, _ := GenerateKey(rand.Reader)
	privKey2, _ := GenerateKey(rand.Reader)
	publicKey := privKey.PublicKey
	publicKey2 := privKey2.PublicKey

	i := big.NewInt(871)
	sig, err := privKey.SignBreak(i, publicKey2)
	if err != nil {
		t.Fatal(err)
	}

	iMinus1 := big.NewInt(1)
	iMinus1.Sub(i, iMinus1)
	err = VerifyPeriods([]PublicKey{publicKey, publicKey2}, sig, []big.Int{*iMinus1, *i})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAggregateSameSigner(t *testing.T) {
	privKey, _ := GenerateKey(rand.Reader)
	privKey2, _ := GenerateKey(rand.Reader)
	publicKey := privKey.PublicKey
	publicKey2 := privKey2.PublicKey

	iMinus1 := big.NewInt(513)
	m := 25
	sigs := make([][]byte, m)
	one := big.NewInt(1)
	var epoch big.Int
	epoch.Set(iMinus1)
	for j := 0; j < m-1; j++ {
		epoch.Add(&epoch, one)
		s, _ := privKey.SignNoBreak(&epoch)
		sigs[j] = s
	}
	epoch.Add(&epoch, one)
	s, err := privKey.SignBreak(&epoch, publicKey2)
	if err != nil {
		t.Fatal(err)
	}
	sigs[m-1] = s

	sig := AggregateSignatures(sigs)
	err = VerifyPeriods([]PublicKey{publicKey, publicKey2}, sig, []big.Int{*iMinus1, epoch})
	if err != nil {
		t.Fatal(err)
	}
}

func TestAggregate(t *testing.T) {
	n := 10
	runs := 3
	perRun := 5

	// prepare keys
	privKeys := make([][]PrivateKey, runs+1)
	pubKeys := make([][]PublicKey, runs+1)
	apks := make([]PublicKey, runs+1)
	for j := 0; j < runs+1; j++ {
		privKeys[j] = make([]PrivateKey, n)
		pubKeys[j] = make([]PublicKey, n)
		for i := 0; i < n; i++ {
			privKey, _ := GenerateKey(rand.Reader)
			privKeys[j][i] = *privKey
			pubKeys[j][i] = privKey.PublicKey
			apks[j].A.Add(&apks[j].A, &privKey.PublicKey.A)
		}
	}

	// prepare signatures
	one := big.NewInt(1)
	epoch := big.NewInt(235)
	breaks := make([]big.Int, runs+1)
	breaks[0].Set(epoch)
	sigs := make([][]byte, n*runs*perRun)
	for j := 0; j < runs; j++ {
		for i := 0; i < perRun-1; i++ {
			epoch.Add(epoch, one)
			for k := 0; k < n; k++ {
				s, _ := privKeys[j][k].SignNoBreak(epoch)
				sigs[j*perRun*n+i*n+k] = s
			}
		}
		epoch.Add(epoch, one)
		breaks[j+1].Set(epoch)
		for k := 0; k < n; k++ {
			s, _ := privKeys[j][k].SignBreak(epoch, apks[j+1])
			sigs[j*perRun*n+(perRun-1)*n+k] = s
		}
	}

	sig := AggregateSignatures(sigs)
	err := VerifyPeriods(apks, sig, breaks)
	if err != nil {
		t.Fatal(err)
	}
}
