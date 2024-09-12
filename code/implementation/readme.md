# Signature Scheme for Light Client Consensus Synchronization
This is a proof of concept implementation of the light client system described in the PDF report.
The idea is that validators add a special signature to reconfiguration blocks (validator set changes), issued at the end of every epoch (ie, a fixed number of blocks).
The signature depends on whether, post-reconfiguration, there remains *the same longest running quorum* (authorized subset, eg, $2f+1$ validators that were active the longest).
This is the case if only few validators are changed by the reconfiguration--the common case in permissioned systems and often in proof of stake.
If the longest running quorum changes, we call this epoch a break point and the end-of-epoch reconfiguration is signed with the `SignBreak` function.
Otherwise, validators sign with `SignNoBreak`.

When receiving a light client request, a full node uses `AggregateSignatures` on the signatures over all epochs since the client's starting epoch from all validators of the longest running quora.
Then, the light client uses `VerifyPeriods` to verify the aggregated signature and the light client stores the new quorum aggregated public key.
This key could be used to subsequently verify a state digest of the blockchain (this is not implemented here since it is deployment specific).

## `signature.go`
In total, the signature scheme implemented in `signature.go` has the following algorithms.
```
func GenerateKey(rand io.Reader) (*PrivateKey, error)

func (privKey *PrivateKey) SignNoBreak(i *big.Int) ([]byte, error)

func (privKey *PrivateKey) SignBreak(i *big.Int, nextApk PublicKey) ([]byte, error)

func AggregateSignatures(sigs [][]byte) []byte

func VerifyPeriods(keys []PublicKey, sigBin []byte, breaks []big.Int) error
```

## `lightclient.go`
Included in `lightclient.go` is a light client implementation using the signature scheme as described.
It serves as an example--specifics required for deployment depend on the overall system in which the light client system is used.
In particular, the examples focuses only on obtaining the latest quorum's apk, it omits verifying the latest block header with that apk (as the header would, of course, depend on the specific blockchain and is out of scope of this generic example).
The main definitions are:
```
func NewLightClient(genesisQuorumApk PublicKey) *LightClient

type LightClient struct {
	Epoch           *big.Int
	QuorumApk       PublicKey
}

func (lc *LightClient) Update(prover Prover) error
```
After calling `Update()` and if the received proof is valid, `QuorumApk` and `Epoch` will have been updated.
The prover must provide one method, returning a proof as described in the PDF.
```
type Prover interface {
	Update(big.Int) (LightClientProof, error)
}

type LightClientProof struct {
	NextApks   []PublicKey
	Signature  []byte
	Breaks     []big.Int
}
```

## Functional tests
The code's proper functioning can be tested with `go test -v` from the `code/implementation` directory.
This command executed all test cases in the `*_test.go` files which ensure the functions' correctness.
If all goes well, the output should be `PASS`.
Note that the test cases `TestAggregate` from `signature_test.go` and `TestUpdate` from `lightclient_test.go` demonstrate the most important behaviors of the functions herein, namely signature aggregation and light client cross-epoch updating.
