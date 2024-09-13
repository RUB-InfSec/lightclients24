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
