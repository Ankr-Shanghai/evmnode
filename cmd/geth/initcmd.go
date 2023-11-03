package main

import (
	"encoding/json"
	"os"

	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

func initGenesis(ctx *cli.Context) error {
	if ctx.Args().Len() != 1 {
		utils.Fatalf("need genesis.json file as the only argument")
	}
	genesisPath := ctx.Args().First()
	if len(genesisPath) == 0 {
		utils.Fatalf("invalid path to genesis file")
	}
	file, err := os.Open(genesisPath)
	if err != nil {
		utils.Fatalf("Failed to read genesis file: %v", err)
	}
	defer file.Close()

	genesis := new(core.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		utils.Fatalf("invalid genesis file: %v", err)
	}

	chaindb, err := OpenDatabase(ctx)
	if err != nil {
		utils.Fatalf("open chaindb failed: %v", err)
	}
	defer chaindb.Close()

	triedb := utils.MakeTrieDatabase(ctx, chaindb, false, false)
	defer triedb.Close()

	_, hash, err := core.SetupGenesisBlock(chaindb, triedb, genesis)
	if err != nil {
		utils.Fatalf("Failed to write genesis block: %v", err)
	}
	log.Info("Successfully wrote genesis state", "database", "chainkv", "hash", hash)
	return nil
}
