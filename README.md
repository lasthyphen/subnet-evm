# EVM Based Enterprise Consortia Chain

Dijets is a network composed of multiple blockchains.
Each blockchain is an instance of a Virtual Machine (VM), much like an object in an object-oriented language is an instance of a class.
That is, the VM defines the behavior of the blockchain.

ECC-EVM Engine is the Virtual Machine (VM) that defines the Subnet Contract Chains. ECC-EVM Engine is a simplified version of [Coreth VM (Utility Chain)](https://github.com/lasthyphen/utilitychain).

This chain implements the Ethereum Virtual Machine and supports Solidity smart contracts as well as most other Ethereum client functionality.

## Building

The ECC-EVM Engine runs in a separate process from the main DIJETSNODE process and communicates with it over a local gRPC connection.

## API

The ECC-EVM Engine supports the following API namespaces:

- `eth`
- `personal`
- `txpool`
- `debug`

Only the `eth` namespace is enabled by default.

## Compatibility

The ECC-EVM Engine is compatible with almost all Ethereum tooling, including Remix, Metamask and Truffle.

## Differences Between ECC-EVM Engine and Coreth

- Added configurable fees and gas limits in genesis
- Merged Dijets hardforks into the single "ECC-EVM Engine" hardfork
- Removed Atomic Txs and Shared Memory
- Removed Multicoin Contract and State

## Block Format

To support these changes, there have been a number of changes to the SubnetEVM block format compared to what exists on the Utility Chain and Ethereum. Here we list the changes to the block format as compared to Ethereum.

### Block Header

- `BaseFee`: Added by EIP-1559 to represent the base fee of the block (present in Ethereum as of EIP-1559)
- `BlockGasCost`: surcharge for producing a block faster than the target rate

## Create an EVM Subnet on a Local Network

### Clone Subnet-evm

First install Go 1.18.1 or later. Follow the instructions [here](https://golang.org/doc/install). You can verify by running `go version`.

Set `$GOPATH` environment variable properly for Go to look for Go Workspaces. Please read [this](https://go.dev/doc/gopath_code) for details. You can verify by running `echo $GOPATH`.

As a few software will be installed into `$GOPATH/bin`, please make sure that `$GOPATH/bin` is in your `$PATH`, otherwise, you may get error running the commands below.

Download the `subnet-evm` repository into your `$GOPATH`:

```sh
cd $GOPATH
mkdir -p src/github.com/lasthyphen
cd src/github.com/lasthyphen
git clone git@github.com:lasthyphen/subnet-evm.git
cd subnet-evm
```

This will clone and checkout to `master` branch.

### Run Local Network

To run a local network, it is recommended to use the [dijets-cli](https://github.com/lasthyphen/dijets-cli) to set up an instance of Subnet-EVM on a local Dijets Network.
