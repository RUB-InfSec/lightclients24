# PoPoS baseline
This baseline is a slightly modified (as stated in our paper's Section 6) version of the [published](https://doi.org/10.4230/LIPIcs.AFT.2023.14) PoPoS codebase.
We describe how we launched the PoPoS experiments using Docker on the server machine.
Here, the server machine launches 8 parallel processes and the client machine uses SSH tunnelling to connect to each one.
This is because, in contrast to our scheme, PoPoS is a multi-server protocol.
In these experiments, each data point (chain length) requires manual intervention.
