# CSSV baseline
This baseline is taken from the CSSV [codebase](https://github.com/w3f/apk-proofs).
In lieu of a client-server implementation, we only perform local experiments for CSSV.
This entails using CSSV's provided `recursive.rs` program, with a minor modification, on a single machine.
Hence, the CSSV experiment has no networking component like the other experiments.
The only modification to `recursive.rs` is that we ensure that the same (sufficient) number of validators sign each committee change, in contrast to choosing a random subsample.
This is done for consistency with the other experiments.
