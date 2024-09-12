#!/usr/bin/env python
"""Plotting script

This script assumes that the experiments for
- our scheme,
- PoPoS,
- CSSV
have been performed as explained in the main Readme file.
The output files are assumed to have been renamed, as explained in the main Readme, and placed in the `plots` directory (from which this script is executed).
"""
import os.path
import json
import re
import numpy as np
import matplotlib.pyplot as plt


def nanoseconds(timestamp):
    if timestamp.endswith('ns'):
        return int(timestamp[:-2])
    elif timestamp.endswith('µs'):
        return int(1e3*float(timestamp[:-2]))
    elif timestamp.endswith('ms'):
        return int(1e6*float(timestamp[:-2]))
    elif timestamp.endswith('s'):
        if 'm' in timestamp:
            idx = timestamp.index('m')
            t = 60*float(timestamp[:idx])+float(timestamp[(idx+1):-1])
        else:
            t = float(timestamp[:-1])
        return int(1e9*t)
    else:
        print('could not convert timestamp', timestamp)
        return 0


def plot_data2(x, y, label, color='black'):
    plt.plot(x, y, label=label, marker='.', color=color)


def plot_data3(x, y, color='black'):
    # to extrapolate missing data
    plt.plot(x, y, marker='.', color=color, linestyle=':')


def numpy_fillna(data):
    # function to fill in missing values if we only did a partial popos experiment
    # from https://stackoverflow.com/a/32043366
    lens = np.array([len(i) for i in data])
    mask = np.arange(lens.max()) < lens[:, None]
    out = np.zeros(mask.shape)
    out[mask] = np.concatenate(data)
    return out


if __name__ == "__main__":
    directory = '.'
    plot_proof_size = True
    plot_latency = True
    plot_prover = True

    # Parse *our scheme's* results
    with open(os.path.join(directory, 'ours_128_wan_fullnode.log'), 'r') as f:
        fullnode = f.readlines()
    with open(os.path.join(directory, 'ours_128_wan_results.log'), 'r') as f:
        results = f.readlines()

    fullnode_comp_from_scratch = {}
    fullnode_comp_multisigs = {}

    regex = re.compile(r'.*\[chain length (\d+)\] served (\w+) in (.*)')
    for line in fullnode:
        ma = regex.match(line)
        if ma is None:
            continue
        m, method, time = ma.groups()
        if method == "UpdateRedundant":
            current_dict = fullnode_comp_from_scratch
        elif method == "UpdateMulti":
            current_dict = fullnode_comp_multisigs
        if int(m) not in current_dict:
            current_dict[int(m)] = []
        current_dict[int(m)].append(nanoseconds(time))

    client_latency_baseline = {}
    client_latency_from_scratch = {}
    client_latency_multisigs = {}

    regex1 = re.compile(r'.*starting benchmarks for Prover(\d+).(\w+)')
    regex2 = re.compile(r'.*updated light client in (.*); local computation: (.*)')
    regex3 = re.compile(r'.*dummy request for (\d+) bytes took (.*)')
    current_dict = client_latency_baseline
    current_logm = 0
    for line in results:
        if 'network latency baseline estimate' in line:
            current_dict = client_latency_baseline
            continue
        ma = regex1.match(line)
        if ma is not None:
            logm, method = ma.groups()
            current_logm = int(logm)
            if method == "UpdateRedundant":
                current_dict = client_latency_from_scratch            
            elif method == "UpdateMulti":
                current_dict = client_latency_multisigs
            if 2**current_logm not in current_dict:
                current_dict[2**current_logm] = []
            continue
        ma = regex2.match(line)
        if ma is not None:
            timeTotal, timeLocal = ma.groups()
            current_dict[2**current_logm].append([nanoseconds(timeTotal), nanoseconds(timeLocal)])
        ma = regex3.match(line)
        if ma is not None:
            length, time = ma.groups()
            if length not in current_dict:
                current_dict[length] = []
            current_dict[length].append(nanoseconds(time))
            continue


    # Parse PoPoS results
    popos = []
    popos_fullnodes = []
    regex = re.compile(r'\d+\s\".*\"\s(\d+\.\d+.*)')
    for i in range(1, 19):
        try:
            f = open(os.path.join(directory, f'popos_128_wan_results_{2**i}.json'), 'r')
        except FileNotFoundError:
            pass
        else:
            with f:
                popos += json.load(f)
        fullnodes = []
        for j in range(8):
            try:
                f = open(os.path.join(directory, f'popos_128_wan_fullnode_{j}_{2**i}'), 'r')
            except FileNotFoundError:
                continue
            else:
                with f:
                    total = 0
                    length = 0
                    for line in f:
                        ma = regex.match(line)
                        if ma is not None:
                            time, = ma.groups()
                            total += nanoseconds(time)
                            length += 1
                    fullnodes.append(total/length)
        popos_fullnodes.append(fullnodes)


    # Parse CSSV results
    with open(os.path.join(directory, 'cssv_128.txt'), 'r') as f:
        cssv = f.readlines()

    cssv_prover = [0]
    cssv_verifier = [0]
    regex1 = re.compile(r'End:\s*Helper aggregated 85 individual signatures on the same commitment and generates accountable light client proof of them (\d+\.\d+.*)')
    regex2 = re.compile(r'End:\s*Light client verifies light client proof for 85 signers \.*(\d+\.\d+.*)')
    regex3 = re.compile(r'Era (\d+)')
    nextPowerOfTwo = 1
    append = False
    for line in cssv:
        ma = regex3.match(line)
        if ma is not None:
            epoch, = ma.groups()
            if int(epoch) == nextPowerOfTwo+1:
                append = True
                nextPowerOfTwo *= 2
            else:
                append = False
        ma = regex1.match(line)
        if ma is not None:
            time, = ma.groups()
            if append:
                if not cssv_prover:
                    cssv_prover.append(nanoseconds(time))
                else:
                    cssv_prover.append(cssv_prover[-1]+nanoseconds(time))
            else:
                cssv_prover[-1] += nanoseconds(time)
            continue
        ma = regex2.match(line)
        if ma is not None:
            time, = ma.groups()
            if append:
                if not cssv_verifier:
                    cssv_verifier.append(nanoseconds(time))
                else:
                    cssv_verifier.append(cssv_verifier[-1]+nanoseconds(time))
            else:
                cssv_verifier[-1] += nanoseconds(time)
    cssv_prover = np.array(cssv_prover)
    cssv_verifier = np.array(cssv_verifier)


    # Generate plots
    if plot_proof_size:
        # Ours
        allX = 2**np.arange(1,21)
        y = [76 * np.ceil(i/2000) for i in allX]
        plot_data2(allX, y, 'Our solution', 'black')

        # CSSV
        y = 976*allX
        plot_data2(allX, y, 'CSSV', 'blue')

        # PoPoS
        ar = np.array([[p['chainSize'], p['bytesDownloaded']] for p in popos])
        ar.sort(axis=0)
        x = np.unique(ar[:, 0])
        # group popos data by unique "chainSize"s
        # from https://stackoverflow.com/questions/38013778/is-there-any-numpy-group-by-function
        y = np.split(ar[:,1], np.unique(ar[:, 0], return_index=True)[1][1:])
        y = np.array([np.mean(data) for data in y])
        plot_data2(x, y, 'PoPoS', 'green')
        # extrapolate
        missing = allX[np.where(allX >= x[-1])]
        plot_data3(missing, [y[-1]]*len(missing), 'green')

        plt.xscale('log', base=2)
        plt.yscale('log')
        plt.xlabel('m (epochs)')
        plt.ylabel('Communication (bytes)')
        plt.legend()
        plt.savefig('eval_communication.pdf')
        plt.close()


    if plot_latency:
        # Ours
        keys = sorted(client_latency_from_scratch.keys())[1:]
        x = np.array(keys)
        allX = x
        y = np.array([np.array(client_latency_from_scratch[k])[:, 0].mean() for k in keys])/1e9
        plot_data2(x, y, 'Our solution', 'black')

        # Ours, precomp. 1 block
        keys = sorted(client_latency_multisigs.keys())[1:]
        x = np.array(keys)
        y = np.array([np.array(client_latency_multisigs[k])[:, 0].mean() for k in keys])/1e9
        plot_data2(x, y, 'Precomp. 1 block', 'red')

        # CSSV
        y = (np.array(cssv_prover)+np.array(cssv_verifier))/1e9
        x = 2**np.arange(1, len(y)+1)
        plot_data2(x, y, 'CSSV', 'blue')

        # PoPoS
        ar = np.array([[p['chainSize'], p['timeToSync']] for p in popos])
        ar.sort(axis=0)
        x = np.unique(ar[:, 0])
        # group popos data by unique "chainSize"s
        # from https://stackoverflow.com/questions/38013778/is-there-any-numpy-group-by-function
        y = np.split(ar[:,1], np.unique(ar[:, 0], return_index=True)[1][1:])
        y = np.array([np.array(data).mean() for data in y])/1e3
        plot_data2(x, y, 'PoPoS', 'green')
        # extrapolate
        missing = allX[np.where(allX >= x[-1])]
        plot_data3(missing, [y[-1]]*len(missing), 'green')

        plt.xscale('log', base=2)
        plt.yscale('log')
        plt.xlabel('m (epochs)')
        plt.ylabel('Time (s)')
        plt.legend()
        plt.savefig('eval_latency_log.pdf')
        plt.close()


    if plot_prover:
        # Ours
        keys = sorted(fullnode_comp_from_scratch.keys())[1:]
        allX = np.array(keys)
        x = allX
        y = np.array([np.array(fullnode_comp_from_scratch[k]).mean() for k in keys])/1e9
        plot_data2(x, y, 'Our solution', 'black')

        # Ours, precomp. 1 block
        keys = sorted(fullnode_comp_multisigs.keys())[1:]
        x = np.array(keys)
        y = np.array([np.array(fullnode_comp_multisigs[k]).mean() for k in keys])/1e9
        plot_data2(x, y, 'Precomp. 1 block', 'red')

        # CSSV
        y = np.array(cssv_prover)[1:]/1e9
        x = 2**np.arange(1, len(y)+1)
        plot_data2(x, y, 'CSSV', 'blue')

        # PoPoS
        x = 2**np.arange(1, len(popos_fullnodes)+1)
        y = numpy_fillna(popos_fullnodes).mean(axis=1)/1e9
        plot_data2(x, y, 'PoPoS', 'green')
        # extrapolate
        missing = allX[np.where(allX >= x[-1])]
        plot_data3(missing, [y[-1]]*len(missing), 'green')

        plt.xscale('log', base=2)
        plt.yscale('log')
        plt.xlabel('m (epochs)')
        plt.ylabel('Time (s)')
        plt.legend()
        plt.savefig('eval_fullnode.pdf')
        plt.close()
