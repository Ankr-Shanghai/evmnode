package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/urfave/cli/v2"
)

var app = &cli.App{}

func init() {
	app.Name = "geth"
	app.Usage = "EVM node command line interface"
	app.Copyright = "Copyright 2013-2023 The go-ethereum/BSC/Ankr Authors"
	app.Commands = []*cli.Command{
		initCmd,
		startCmd,
		importCmd,
		rpccmd,
		versionCommand,
	}
	app.Flags = []cli.Flag{
		utils.Backend,
		utils.DbHost,
		utils.DbPort,
	}
	// app.Flags = append(app.Flags, debug.Flags...)

	sort.Sort(cli.CommandsByName(app.Commands))

}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(-1)
	}
}
