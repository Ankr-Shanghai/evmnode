package main

import (
	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/urfave/cli/v2"
)

var (
	initCmd = &cli.Command{
		Name:      "init",
		Usage:     "initialize genesis block",
		ArgsUsage: "<genesis path>",
		Action:    initGenesis,
		Description: `
       The init command initializes a new genesis block and definition for the network.
       This is a destructive action and changes the network in which you will be
       participating.
       It expects the genesis file as argument.`,
	}

	startCmd = &cli.Command{
		Name:  "start",
		Usage: "boot main service",
		Flags: []cli.Flag{
			utils.SvcHost,
			utils.SvcPort,
		},
		Action: start,
	}

	importCmd = &cli.Command{
		Name:   "import",
		Usage:  "import missing blocks",
		Action: restore,
		Flags: []cli.Flag{
			utils.SnapPath,
			utils.SnapEngine,
		},
	}

	rpccmd = &cli.Command{
		Name:   "rpc",
		Usage:  "boot rpc service",
		Action: rpcstart,
		Flags: []cli.Flag{
			utils.SvcHost,
			utils.SvcPort,
		},
	}
)
