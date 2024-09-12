// Custom signature scheme for blockchain light clients
//
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

// Package lightclientsignature provides a transitive signature scheme for
// blockchain light client synchronization, on the bls12-381 elliptic curve.
package lightclientsignature
