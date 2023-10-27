package main

import (
	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var (
	ethereum *eth.Ethereum
	err      error
)

func newBlockChain(ctx *cli.Context) {
	// addr := ctx.String(utils.DbHost.Name) + ":" + ctx.String(utils.DbPort.Name)
	// db, err := pika.New(addr)
	// if err != nil {
	// 	utils.Fatalf("Failed to open database: %v", err)
	// }

	db, err := rawdb.NewPebbleDBDatabase("data", 1024, 128, "", false)
	if err != nil {
		utils.Fatalf("Failed to open database: %v", err)
	}

	cfg := &ethconfig.Defaults
	cfg.NoPruning = true
	cfg.TriesVerifyMode = core.FullVerify

	ethereum = eth.NewEthereum(db, cfg)

	log.Info("create blockchain success")
}
