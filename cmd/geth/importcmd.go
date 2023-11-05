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
	engine := ctx.String(utils.SnapEngine.Name)
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

	idx := rawdb.ReadHeadHeader(chainDb).Number.Uint64() + 1
	snapBlockNumber := rawdb.ReadHeadBlock(chainDb).Number().Uint64()
	log.Info("restore", "idx", idx, "snapBlockNumber", snapBlockNumber)

	//
	for ; idx <= snapBlockNumber; idx++ {
		block := rawdb.ReadBlock(chainDb, rawdb.ReadCanonicalHash(chainDb, idx), idx)
		ethereum.BlockChain().InsertChain([]*types.Block{block})
	}
	log.Info("restore done", "current block", idx)
	return nil
}
