package main

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/pkg/source"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sunvim/utils/grace"
	"github.com/urfave/cli/v2"
)

func start(ctx *cli.Context) error {
	// init backend client to take new block
	source.InitBackendClient(ctx)

	_, gs := grace.New(context.Background())

	// create blockchain
	newBlockChain(ctx)

	gs.Register(func() error {
		ethereum.Stop()
		return nil
	})

	gs.RegisterService("evm", func(c context.Context) error {
		srv := rpc.NewServer()

		apis := getAllAPIs(ethereum.APIBackend)

		for _, api := range apis {
			if err := srv.RegisterName(api.Namespace, api.Service); err != nil {
				log.Error("rpc.RegisterName", "err", err)
			}
		}
		handler := adaptor.HTTPHandler(srv)

		svc := fiber.New(fiber.Config{
			Prefork:               false,
			ServerHeader:          "Ankr team",
			DisableStartupMessage: true,
			StreamRequestBody:     true,
			BodyLimit:             500 * 1024 * 1024,
		})

		svc.Use(recover.New())
		svc.Post("/", handler)
		svc.Post("/block", missingBlockHandler)

		addr := ctx.String(utils.SvcHost.Name) + ":" + ctx.String(utils.SvcPort.Name)
		log.Info("evm service boot", "entrypoint", addr)
		if err := svc.Listen(addr); err != nil {
			log.Error("evm service boot", "err", err)
			return err
		}
		return nil
	})

	gs.Wait()
	return nil
}

func missingBlockHandler(ctx *fiber.Ctx) error {
	body := ctx.Body()
	var blks []json.RawMessage
	err := json.Unmarshal(body, &blks)
	if err != nil {
		log.Error("missingBlockHandler", "err", err)
		return err
	}
	blocks := make(types.Blocks, 0, len(blks))
	for _, blk := range blks {
		block := formatBlock(blk)
		blocks = append(blocks, block)
	}
	ethereum.BlockChain().InsertChain(blocks)
	return nil
}

func formatBlock(raw json.RawMessage) *types.Block {
	var head *types.Header
	if err := json.Unmarshal(raw, &head); err != nil {
		log.Error("head unmarshal err", "err", err)
		return nil
	}
	var body rpcBlock
	if err := json.Unmarshal(raw, &body); err != nil {
		log.Error("body unmarshal err", "err", err)
		return nil
	}
	txs := make([]*types.Transaction, len(body.Transactions))
	for i, tx := range body.Transactions {
		txs[i] = tx.tx
	}
	return types.NewBlockWithHeader(head).WithBody(txs, nil).WithWithdrawals(body.Withdrawals)
}

type rpcBlock struct {
	Hash         common.Hash         `json:"hash"`
	Transactions []rpcTransaction    `json:"transactions"`
	UncleHashes  []common.Hash       `json:"uncles"`
	Withdrawals  []*types.Withdrawal `json:"withdrawals,omitempty"`
}

type rpcTransaction struct {
	tx *types.Transaction
	txExtraInfo
}

func (tx *rpcTransaction) UnmarshalJSON(msg []byte) error {
	if err := json.Unmarshal(msg, &tx.tx); err != nil {
		return err
	}
	return json.Unmarshal(msg, &tx.txExtraInfo)
}

type txExtraInfo struct {
	BlockNumber *string         `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash    `json:"blockHash,omitempty"`
	From        *common.Address `json:"from,omitempty"`
}

func takeBlocks(wg *sync.WaitGroup, client *ethclient.Client, from, to uint64, chanBlocks chan<- types.Blocks) {
	defer wg.Done()
	for i := from; i <= to; i++ {
		block, err := client.BlockByNumber(context.Background(), big.NewInt(int64(i)))
		if err != nil {
			log.Error("takeBlocks", "err", err)
			os.Exit(-1)
		}
		chanBlocks <- types.Blocks{block}
	}
}

func consumeBlocks(wg *sync.WaitGroup, chanBlocks <-chan types.Blocks) {
	defer wg.Done()
	for {
		select {
		case blks := <-chanBlocks:
			ethereum.BlockChain().InsertChain(blks)
		}
	}
}
