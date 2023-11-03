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

	option := rawdb.OpenOptions{
		Type:             "chainkv",
		Host:             ctx.String(utils.DbHost.Name),
		Port:             ctx.String(utils.DbPort.Name),
		DisableFreeze:    true,
		PruneAncientData: false,
		ReadOnly:         false,
	}

	chaindb, err := rawdb.Open(option)
	if err != nil {
		log.Error("open chaindb failed", "err", err)
	}

	cfg := &ethconfig.Defaults
	cfg.NoPruning = true
	cfg.TriesVerifyMode = core.FullVerify

	ethereum = eth.NewEthereum(chaindb, cfg)

	log.Info("create blockchain success")
}
