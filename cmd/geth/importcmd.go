package main

import (
	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func restore(ctx *cli.Context) error {
	// create blockchain
	newBlockChain(ctx)

	snapPath := ctx.String(utils.SnapPath.Name)
	engine := ctx.String(utils.Engine.Name)
	var (
		chainDb ethdb.Database
		option  rawdb.OpenOptions
		err     error
	)

	option.Directory = snapPath
	option.Type = engine
	option.ReadOnly = true

	chainDb, err = rawdb.Open(option)
	if err != nil {
		return err
	}

	localBlockNuber := ethereum.BlockChain().CurrentHeader().Number.Uint64()
	snapBlockNumber := rawdb.ReadHeadBlock(chainDb).Number().Uint64()
	log.Info("restore", "localBlockNuber", localBlockNuber, "snapBlockNumber", snapBlockNumber)

	//
	for {
		if localBlockNuber > snapBlockNumber {
			break
		}
		block := rawdb.ReadBlock(chainDb, rawdb.ReadCanonicalHash(chainDb, localBlockNuber), localBlockNuber)
		ethereum.BlockChain().InsertChain([]*types.Block{block})
		localBlockNuber++
	}
	log.Info("restore done", "current block", localBlockNuber)
	return nil
}
