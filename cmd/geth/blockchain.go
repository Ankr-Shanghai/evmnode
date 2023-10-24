package main

import (
	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb/pika"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var (
	ethereum *eth.Ethereum
	err      error
)

func newBlockChain(ctx *cli.Context) {
	addr := ctx.String(utils.DbHost.Name) + ":" + ctx.String(utils.DbPort.Name)
	db, err := pika.New(addr)
	if err != nil {
		utils.Fatalf("Failed to open database: %v", err)
	}

	ethereum = eth.NewEthereum(db, &ethconfig.Defaults)

	log.Info("create blockchain success")
}
