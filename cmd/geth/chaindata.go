package main

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/urfave/cli/v2"
)

func chaindata(ctx *cli.Context) error {
	snapPath := ctx.String(utils.SnapPath.Name)
	engine := ctx.String(utils.SnapEngine.Name)
	var (
		chaindb ethdb.Database
		option  rawdb.OpenOptions
		err     error
	)

	option.Directory = snapPath
	option.Type = engine
	option.ReadOnly = true

	chaindb, err = rawdb.Open(option)
	if err != nil {
		return err
	}
	defer chaindb.Close()

	// get the latest block number
	blockNumber := rawdb.ReadHeadHeader(chaindb).Number.Uint64()

	if blockNumber > 10 {
		blockNumber = 10
	}

	var idx uint64 = 1

	for idx = 1; idx < blockNumber; idx++ {
		block := rawdb.ReadBlock(chaindb, rawdb.ReadCanonicalHash(chaindb, idx), idx)
		fmt.Printf("block Number: %d \n", block.NumberU64())
		if len(block.Transactions()) != 0 {
			for _, tx := range block.Transactions() {
				fmt.Printf("tx hash: %s \n", tx.Hash().Hex())
				fmt.Printf("tx to: %s \n", tx.To().Hex())
			}
		}
		fmt.Println(strings.Repeat("==", 20))
	}

	return nil
}
