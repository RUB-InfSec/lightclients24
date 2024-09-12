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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"errors"
	"io"
	"math/big"
	"runtime"

	curve "github.com/consensys/gnark-crypto/ecc/bls12-377"
	"github.com/consensys/gnark-crypto/ecc/bls12-377/fp"
	"github.com/consensys/gnark-crypto/ecc/bls12-377/fr"
)

const (
	sizeFr         = fr.Bytes
	sizeFp         = fp.Bytes
	sizePublicKey  = sizeFp
	sizePrivateKey = sizeFr + sizePublicKey
	sizeSignature  = 2 * sizeFp
)

var order = fr.Modulus()

// PublicKey represents a public key
type PublicKey struct {
	A curve.G1Affine
}

// PrivateKey represents a private key
type PrivateKey struct {
	PublicKey PublicKey
	scalar    [sizeFr]byte // secret scalar, in big Endian
}

// Signature represents a signature or aggregated signature
type Signature struct {
	S curve.G2Affine
}

var one = new(big.Int).SetInt64(1)

// randFieldElement returns a random element of the order of the given
// curve using the procedure given in FIPS 186-4, Appendix B.5.1.
func randFieldElement(rand io.Reader) (k *big.Int, err error) {
	b := make([]byte, fr.Bits/8+8)
	_, err = io.ReadFull(rand, b)
	if err != nil {
		return
	}

	k = new(big.Int).SetBytes(b)
	n := new(big.Int).Sub(order, one)
	k.Mod(k, n)
	k.Add(k, one)
	return
}

// GenerateKey generates a public and private key pair.
func GenerateKey(rand io.Reader) (*PrivateKey, error) {

	k, err := randFieldElement(rand)
	if err != nil {
		return nil, err
	}
	_, _, g, _ := curve.Generators()

	privateKey := new(PrivateKey)
	k.FillBytes(privateKey.scalar[:sizeFr])
	privateKey.PublicKey.A.ScalarMultiplication(&g, k)
	return privateKey, nil
}

type zr struct{}

// Read replaces the contents of dst with zeros. It is safe for concurrent use.
func (zr) Read(dst []byte) (n int, err error) {
	for i := range dst {
		dst[i] = 0
	}
	return len(dst), nil
}

var zeroReader = zr{}

const (
	aesIV = "gnark-crypto IV." // must be 16 chars (equal block size)
)

func nonce(privateKey *PrivateKey, hash []byte) (csprng *cipher.StreamReader, err error) {
	// This implementation derives the nonce from an AES-CTR CSPRNG keyed by:
	//
	//    SHA2-512(privateKey.scalar ∥ entropy ∥ hash)[:32]
	//
	// The CSPRNG key is indifferentiable from a random oracle as shown in
	// [Coron], the AES-CTR stream is indifferentiable from a random oracle
	// under standard cryptographic assumptions (see [Larsson] for examples).
	//
	// [Coron]: https://cs.nyu.edu/~dodis/ps/merkle.pdf
	// [Larsson]: https://web.archive.org/web/20040719170906/https://www.nada.kth.se/kurser/kth/2D1441/semteo03/lecturenotes/assump.pdf

	// Get 256 bits of entropy from rand.
	entropy := make([]byte, 32)
	_, err = io.ReadFull(rand.Reader, entropy)
	if err != nil {
		return

	}

	// Initialize an SHA-512 hash context; digest...
	md := sha512.New()
	md.Write(privateKey.scalar[:sizeFr]) // the private key,
	md.Write(entropy)                    // the entropy,
	md.Write(hash)                       // and the input hash;
	key := md.Sum(nil)[:32]              // and compute ChopMD-256(SHA-512),
	// which is an indifferentiable MAC.

	// Create an AES-CTR instance to use as a CSPRNG.
	block, _ := aes.NewCipher(key)

	// Create a CSPRNG that xors a stream of zeros with
	// the output of the AES-CTR instance.
	csprng = &cipher.StreamReader{
		R: zeroReader,
		S: cipher.NewCTR(block, []byte(aesIV)),
	}

	return csprng, err
}

// Equal compares 2 public keys
func (pub *PublicKey) Equal(pk *PublicKey) bool {
	pubb := pub.Bytes()
	pkb := pk.Bytes()
	return subtle.ConstantTimeCompare(pubb, pkb) == 1
}

// Public returns the public key associated to the private key.
func (privKey *PrivateKey) Public() PublicKey {
	var pub PublicKey
	pub.A.Set(&privKey.PublicKey.A)
	return pub
}

// ChainHash returns H(i+m-1)/H(i-1)
func ChainHash(i, m *big.Int) (curve.G2Affine, error) {
	dst1 := []byte("0x01")
	epoch := new(big.Int)
	epoch.Sub(i, one)
	hash, err := curve.HashToG2(epoch.Bytes(), dst1)
	if err != nil {
		return curve.G2Affine{}, err
	}
	hash.Neg(&hash)
	epoch.Add(epoch, m)
	_h, err := curve.HashToG2(epoch.Bytes(), dst1)
	if err != nil {
		return curve.G2Affine{}, err
	}
	hash.Add(&hash, &_h)
	return hash, nil
}

// ChainHashKeys returns H(i+m-1, apk)/H(i-1)
func ChainHashKeys(i, m *big.Int, apk PublicKey) (curve.G2Affine, error) {
	dst1 := []byte("0x01")
	dst2 := []byte("0x02")
	epoch := new(big.Int)
	epoch.Sub(i, one)
	hash, err := curve.HashToG2(epoch.Bytes(), dst1)
	if err != nil {
		return curve.G2Affine{}, err
	}
	epoch.Add(epoch, m)
	toHash := append(epoch.Bytes(), apk.Bytes()...)
	_h, err := curve.HashToG2(toHash, dst2)
	if err != nil {
		return curve.G2Affine{}, err
	}
	hash.Neg(&hash)
	hash.Add(&hash, &_h)
	return hash, nil
}

// SignNoBreak computes a signature that signals NO committee change
func (privKey *PrivateKey) SignNoBreak(i *big.Int) ([]byte, error) {
	sk := new(big.Int)
	sk.SetBytes(privKey.scalar[:sizeFr])
	ch, err := ChainHash(i, one)
	if err != nil {
		return nil, err
	}
	var sig Signature
	sig.S.ScalarMultiplication(&ch, sk)
	return sig.Bytes(), nil
}

// SignBreak computes a signature over a committee change
func (privKey *PrivateKey) SignBreak(i *big.Int, nextApk PublicKey) ([]byte, error) {
	sk := new(big.Int)
	sk.SetBytes(privKey.scalar[:sizeFr])
	ch, err := ChainHashKeys(i, one, nextApk)
	if err != nil {
		return nil, err
	}
	var sig Signature
	sig.S.ScalarMultiplication(&ch, sk)
	return sig.Bytes(), nil
}

// VerifyPeriods verifies a (aggregated) signature over a sequence of NextKeys
func VerifyPeriods(apks []PublicKey, sigBin []byte, breaks []big.Int) error {
	Len := len(breaks)
	// breaks[0] is the starting epoch minus one;
	// breaks[1], ..., breaks[Len-2] are break points;
	// breaks[Len-1] is the new epoch minus one;
	// apks has either length Len (if breaks[Len-1] is a break point) or length Len-1;
	// apks[0] is the apk of the starting epoch;
	// apks[i] is the apk to verify the next key apks[i+1].
	if (len(apks) != Len) && (len(apks) != Len-1) {
		return errors.New("wrong length of apks")
	}

	// Deserialize the signature
	var sig curve.G2Affine
	if _, err := sig.SetBytes(sigBin); err != nil {
		return err
	}

	// Prepare the hashes and public keys
	hashes := make([]curve.G2Affine, Len-1)
	keys := make([]curve.G1Affine, Len-1)
	for i := 0; i < Len-1; i++ {
		// ChainHash subtracts 1 from its first argument (so we must add 1 before)
		// and takes a difference as its second argument
		var inc, diff big.Int
		var h curve.G2Affine
		var err error
		inc.Add(&breaks[i], one)
		diff.Sub(&breaks[i+1], &breaks[i])
		if (i == Len-2) && (len(apks) == Len-1) {
			h, err = ChainHash(&inc, &diff)
		} else {
			h, err = ChainHashKeys(&inc, &diff, apks[i+1])
		}
		if err != nil {
			return err
		}
		hashes[i] = h
		keys[i] = apks[i].A
	}

	// Verify the signature
	_, _, g1, _ := curve.Generators()
	g1.Neg(&g1)
	keys = append(keys, g1)
	hashes = append(hashes, sig)
	ok, err := curve.PairingCheck(keys, hashes)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("verification failed")
	}

	return nil
}

// func asBytes(pks []PublicKey) []byte {
// 	b := []byte{}
// 	for _, pk := range pks {
// 		b = append(b, pk.Bytes()...)
// 	}
// 	return b
// }

// AggregateSignatures multiplies signatures
func AggregateSignatures(sigs [][]byte) []byte {
	minPerCPU := 2
	n := runtime.NumCPU()
	if n > len(sigs)/minPerCPU {
		n = 1 + len(sigs)/minPerCPU
	}
	perCPU := len(sigs) / n

	// Goroutines handle consecutive chunks of length `perCPU` (or `perCPU`+1)
	results := make(chan curve.G2Affine)
	leftover := len(sigs) - n*perCPU
	var start, end, offset int
	for i := 0; i < n; i++ {
		start = i*perCPU + offset
		end = start + perCPU
		if leftover > 0 {
			end++
			offset++
			leftover--
		}
		go func(a, b int) {
			var p, temp curve.G2Affine
			for i := a; i < b; i++ {
				temp.Unmarshal(sigs[i])
				p.Add(&p, &temp)
			}
			results <- p
		}(start, end)
	}

	var asig curve.G2Affine
	for i := 0; i < n; i++ {
		received := <-results
		asig.Add(&asig, &received)
	}
	return asig.Marshal()
}
