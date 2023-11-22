package main

import (
	"runtime/debug"

	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/urfave/cli/v2"
)

var (
	ethereum *eth.Ethereum
	err      error
	cfg      *ethconfig.Config
)

func newBlockChain(ctx *cli.Context) {

	chaindb, err := OpenDatabase(ctx)
	if err != nil {
		log.Error("open chaindb failed", "err", err)
		return
	}

	cfg = &ethconfig.Defaults
	cfg.NoPruning = true
	cfg.TriesVerifyMode = core.FullVerify
	cfg.RangeLimit = true
	cfg.NetworkId = 56

	ethereum = eth.NewEthereum(chaindb, cfg)
	ethereum.Start()

	debug.SetMemoryLimit(24 * opt.GiB)

	log.Info("create blockchain success")

	// export state routine start
	startExportState(ctx)
}

func OpenDatabase(ctx *cli.Context) (ethdb.Database, error) {

	var option = rawdb.OpenOptions{
		DisableFreeze:    true,
		PruneAncientData: false,
		ReadOnly:         false,
	}

	switch ctx.String(utils.Engine.Name) {
	case "chainkv":
		option.Type = "chainkv"
		option.Host = ctx.String(utils.DbHost.Name)
		option.Port = ctx.String(utils.DbPort.Name)
		option.Size = ctx.Int(utils.DbSize.Name)
	case "pebble":
		option.Type = "pebble"
		option.Directory = ctx.String(utils.DataDir.Name)
		option.AncientsDirectory = ""
		option.Cache = 4 * 1024 // 1G
		option.Handles = 1024
	case "leveldb":
		option.Type = "leveldb"
		option.Directory = ctx.String(utils.DataDir.Name)
		option.AncientsDirectory = ""
		option.Cache = 4 * 1024 // 1G
		option.Handles = 1024
	}

	chaindb, err := rawdb.Open(option)
	if err != nil {
		log.Error("open chaindb failed", "err", err)
		return nil, err
	}
	return chaindb, nil
}
