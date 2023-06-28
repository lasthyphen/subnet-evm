// (c) 2019-2020, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package main

import (
	"fmt"
	"os"

	"github.com/lasthyphen/dijetsnode/utils/logging"
	"github.com/lasthyphen/dijetsnode/utils/ulimit"
	"github.com/lasthyphen/dijetsnode/version"
	"github.com/lasthyphen/dijetsnode/vms/rpcchainvm"

	"github.com/lasthyphen/subnet-evm/plugin/evm"
)

func main() {
	printVersion, err := PrintVersion()
	if err != nil {
		fmt.Printf("couldn't get config: %s", err)
		os.Exit(1)
	}
	if printVersion {
		fmt.Printf("Subnet-EVM/%s [DIJETSNODE=%s, rpcchainvm=%d]\n", evm.Version, version.Current, version.RPCChainVMProtocol)
		os.Exit(0)
	}
	if err := ulimit.Set(ulimit.DefaultFDLimit, logging.NoLog{}); err != nil {
		fmt.Printf("failed to set fd limit correctly due to: %s", err)
		os.Exit(1)
	}
	rpcchainvm.Serve(&evm.VM{})
}
