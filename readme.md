# ACSAC 2024 Submission 309
This repo contains the initially submitted version of paper 309 at ACSAC 2024, "Practical Light Clients for Committee-Based Blockchains", together with the source code for our implementation and evaluation.
The paper is `acsac_submission_28may24.pdf`.
The `code/implementation` and `code/evaluation` directories allow reproducing the paper's results (Sections 5 and 6) as we explain below.

## Paper abstract
Light clients are gaining increasing attention in the literature since they obviate the need for users to set up dedicated blockchain full nodes. While the literature features a number of light client instantiations, most light client protocols optimize for long offline phases and implicitly assume that the block headers to be verified are signed by highly dynamic validators.
In this paper, we show that (i) most light clients are rarely offline for more than a week, and (ii) validators are unlikely to drastically change in most permissioned blockchains and in a number of permissionless blockchains, such as Cosmos and Polkadot. Motivated by these findings, we propose a novel practical system that optimizes for such realistic assumptions and achieves minimal communication and computational costs for light clients when compared to existing protocols. By means of a prototype implementation of our solution, we show that our protocol achieves a reduction by up to 90 and 40000× (respectively) in end-to-end latency and up to 1000 and 10000× (respectively) smaller proof size when compared to two state-of-the-art light client instantiations from the literature.

## Artifact abstract
This is the artifact submission corresponding to the conditionally accepted paper \#309.
Our paper proposes a light client system for committee-based blockchains that relies on validators signing end-of-epoch committee changes with a novel transitive signature scheme.
Our artifact consists of source code for the signature scheme and for creating and verifying light client proofs.
We also include all the code required to reproduce our client-server evaluation, including the baseline comparisons, which are also depicted in the main submission’s Figure 5.
The artifact, therefore, includes (slightly modified) versions of the two alternative schemes, PoPoS and CSSV.
Our artifact has been tested on a dual 64-core AMD EPYC 7742 machine (as the server) and an 8-core Apple M2 laptop (as the client) over a wide area network.

## Subdirectory structure
```
.
├── readme.md                      /* this file */
├── acsac_submission_28may24.pdf   /* initially submitted paper */
└── code/
    ├── implementation/            /* description and implementation of our scheme */
    └── evaluation/                /* evaluation code; how figure 5 was obtained */
        ├── baselines/
        │   ├── cssv/              /* local cssv evalution */
        │   └── popos/             /* client and server code for popos evaluation */
        ├── full_node/             /* server-side test application for our scheme */
        ├── tester_client/         /* client-side test application for our scheme */
        └── plots/                 /* plotting script for all schemes' evaluation results */
```

## How to use this artifact
If needed, a mirror of this repository is available for download at the following URL: [link](https://1drv.ms/f/s!AsMByznOBlivg0GtR_FFK-djSJIR?e=Un6FDy).

### Implementation
The Go code in `code/implementation` uses the elliptic curve cryptography library gnark-crypto in order to provide an implementation of our cryptographic protocol's basic primitive--the transitive siganture scheme.
The functions in `code/implementation/signature.go` implement the signing and verfying algorithms described and proven secure in our paper.
For more information on the functions provided by our Go code, also see the Readme file in `code/implementation`.

### Evaluation
The `code/evaluation` directory allows reproducing our evaluation, with the outcome being the plots shown in Figure 5.
For this evaluation, we launched test applications (consisting of server and client) for our scheme and for the PoPoS and CSSV baselines in a WAN setting and collected latency measurements.
Our scheme's test application uses the implementation from `code/implementation`.
The PoPoS application is a slightly adapted version (as explained in our paper) of the code released along with the original PoPoS paper.
The same is true for CSSV.
However, for CSSV, no networked client-server implementations were available.
Hence, we only used local benchmarks for CSSV (ie, proof creation and verification are benchmarked on the same machine and there is simply no network latency component in the reported "end-to-end latency" for CSSV).

##### Remark on dummy data generation
Each experiment requires a pre-computation phase where dummy data is generated (for use in the to-be-measured protocol runs).
In our paper we used dummy data of size up to $2^{20}$, which took our powerful server machine several days to generate.
To avoid this runtime for testing purposes, in this repository, default values of $2^{10}$ are used.
On an Apple M1 machine, generating this reduced amount of dummy data took up to 20 minutes for each experiment.
We include instructions for increasing the data size to $2^{20}$ in order to replicate our paper's plots exactly.
Additionally, we provide pre-generated dummy data for the PoPoS experiments (where this process takes the longest) at the following URL: [link](https://1drv.ms/f/s!AsMByznOBlivg0GtR_FFK-djSJIR?e=Un6FDy).

##### Dependencies
We use Docker containers and, especially important for our WAN measurements, SSH tunnelling.
Overall, the server machine should have the following programs available to run all experiments.
- Docker.

The client machine should have the following programs available:
- Go 1.22.
- (Only for the PoPoS baseline:) Node.js 21 with npm, yarn, typescript. Freshly install the latter two packages as
```
npm install -g yarn
npm install -g typescript
```

Additionally, if the experiments are performed on separate machines (in the WAN setting), the client needs to be able to connect with the server over SSH on ports 7890,7770-7777 (ports can be adjusted by modifying the respective `ssh` commands shown below).

#### 1. Experiments for our scheme
##### Server
- From the `code` directory, run the following command.
```
docker run -it -v .:/data -p 7890:7890 golang:1.22
```
- In the container, use the following commands to start the full node.
  For the first time, the full node will take a while to create all dummy data (approximately 15 minutes on an Apple M1 Pro machine).
```
cd /data/evaluation/full_node
go build -o fullnode main.go
./fullnode
```
- Optionally, from outside the container, view the progress logs of the full node as follows:
```
tail -f evaluation/full_node/fullnode.log
```
- After the `tester_client` (see below) is done issuing requests, the full node's logs are available as `fullnode.log` in the `evaluation/full_node` directory.
  To be able to generate the plots, rename `fullnode.log` to `ours_128_wan_fullnode.log` and move it to `evaluation/plots`.
- To benchmark longer dummy chains like in the paper, in line 387 in `evaluation/full_node/main.go` increase the value `1 << 10` to the desired longer chain length.

##### Client
- Copy `nextkeys0.gob` from the server machine's `evaluation/full_node` directory to the client machine's `evaluation/tester_client` directory.
- Compile the `tester_client` application with the following command from `evaluation/tester_client`:
```
go build -o tester_client main.go
```
- If the full node is **not** running on the same machine as the client but is instead reachable at the IP address `<IP-ADDRESS>` (eg, over a WAN), set up SSH port forwarding by running the following command (assuming a working SSH connection from the client machine to the server machine):
```
ssh -fN -L 7890:127.0.0.1:7890 <IP-ADDRESS>
```
- Run the following command to launch the experiments.
  This might take approximately 30 minutes to complete (on an Apple M1 Pro machine).
```
tester_client 127.0.0.1 7890 10 50
```
- After the client is done issuing requests, the client's logs are available as `results.log` in the `evaluation/tester_client` directory.
  To be able to generate the plots, rename `results.log` to `ours_128_wan_results.log` and move it to the server machine's `evaluation/plots` directory.

#### 2. Experiments for PoPoS
##### Server
- **For each** power-of-two chain length `<LENGTH>` from $2^{10} = 1024$ down to $2^1 = 2$:
  - In line 12 in `compose.yml`, set the chain length to `<LENGTH>`.
    If the server machine has less than 8 CPU cores, modify the CPUs assigned to each of the containers in `compose.yml` (`cpuset` fields).
    (The benchmarks in our paper used 16 independent cores per container.)
  - Run the following command.
    The process is complete once all 7 "dishonest" containers output `Server listening on port 3679`.
    The first time, all dummy data is generated, which might take up to 15 minutes on an Apple M2 machine.
```
docker compose build --no-cache && docker compose up
```
  - Issue requests from the client as described below.
- To benchmark for more than $2^{10}$ epochs should be generated, set the desired length(s) in line 30 in `evaluation/baselines/popos/implementation/src/prover/router.ts`.
  Be aware that, in this case, the initial dummy data generation will also take longer.
- For all `<i>` from 1 to 7 and for all `<length>`, move each file `evaluation/baselines/popos/dishonest<i>_data/timer_<length>.log` to `evaluation/plots` and rename it to `popos_128_wan_fullnode_<i>_<length>`.
- To be able to generate the plots, for all `<length>`, move `evaluation/baselines/popos/honest_data/timer_<length>.log` to `evaluation/plots` and rename it to `popos_128_wan_fullnode_0_<length>`.

##### Client
- Run the following command from `evaluation/baselines/popos/implementation`.
```
mkdir results
```
- If the server is **not** running on the same machine as the client but is instead reachable at the IP address `<IP-ADDRESS>` (eg, over a WAN), set up SSH port forwarding by running the following commands (assuming a working SSH connection from the client machine to the server machine).
```
for (( i=7770; i<=7777; i++ ))
do
    ssh -fN -L $i:127.0.0.1:$i <IP-ADDRESS>
done
```
- **For each** power-of-two chain length `<LENGTH>` from $2^{10} = 1024$ down to $2^2 = 2$:
  - In line 17 in `evaluation/baselines/popos/implementation/benchmark/multiple-server.ts`, change the chain length (`size` variable) to `<LENGTH>`.
  - After the *server machine* is set up for this chain length, execute the following commands from `evaluation/baselines/popos/implementation` to start the experiment:
```
yarn install
yarn build
node dist/benchmark/multiple-server.js
```
- To be able to generate the plots, for all `<length>`, move each file `evaluation/baselines/popos/implementation/results/dummy-data-8-128-<length>-100-1.json` to the server machine's `evaluation/plots` directory and rename it to `popos_128_wan_results_<length>.json`.

#### 3. Experiments for CSSV
##### Server
- Start a Docker container with the following command from `evaluation/baselines/cssv`.
```
docker run -it -v .:/code rust:1.77
```
- In the container, execute the following command to launch the local experiment (where light client proofs are created and verified on the same machine).
```
cd /code
cargo run --release --features "parallel print-trace" --example recursive 7 1024 > cssv_128.txt
```
- Optionally, from outside the container, view the progress logs as follows:
```
tail -f cssv_128.txt
```
- To be able to generate the plots, move `evaluation/baselines/cssv/cssv_128.txt` to `evaluation/plots`.

#### 4. Generating the plots
When the experiment outputs are renamed as described above and all moved to `evaluation/plots` on the server machine, run the following commands from `evaluation/plots`.
```
docker build -t lightclientplots .
docker run -it -v .:/data lightclientplots
```

#### Troubleshooting
