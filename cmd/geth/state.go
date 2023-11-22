package main

import (
	"time"

	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/urfave/cli/v2"
)

var (
	FiveHundredW uint64 = 50_000
	OneHundredW  uint64 = 10_000
)

// greater than 500W, export all state per 100W blocks
func startExportState(ctx *cli.Context) {
	if !ctx.Bool(utils.ExportState.Name) {
		return
	}

	log.Info("start export state service")

	chaindb := ethereum.ChainDb()

	var flagNumber uint64 = 0

	go func() {
		ticker := time.Tick(30 * time.Second)
		for range ticker {
			startBlockNumber := ethereum.APIBackend.CurrentHeader().Number.Uint64()
			if startBlockNumber < FiveHundredW {
				continue
			}
			cur := startBlockNumber / OneHundredW

			if cur < flagNumber {
				continue
			}

			flagNumber = cur
			curState := cur * OneHundredW
			curHeader := rawdb.ReadHeader(chaindb, rawdb.ReadCanonicalHash(chaindb, curState), curState)
			trdb := ethereum.BlockChain().TrieDB()

			tr, err := trie.NewStateTrie(trie.StateTrieID(curHeader.Root), trdb)
			if err != nil {
				log.Error("new state trie failed", "err", err)
				continue
			}
			it := trie.NewIterator(tr.MustNodeIterator(nil))

			for it.Next() {
			}

		}
	}()
}
