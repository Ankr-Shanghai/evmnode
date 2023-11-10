package main

import (
	"context"
	"net/http"
	"os"

	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gofiber/fiber/v2"
	"github.com/sunvim/utils/grace"
	"github.com/urfave/cli/v2"
)

var chanBlocks chan types.Blocks

func importstore(ctx *cli.Context) error {
	chanBlocks = make(chan types.Blocks, 128)
	_, gs := grace.New(context.Background())

	// create blockchain
	newBlockChain(ctx)

	gs.Register(func() error {
		ethereum.Stop()
		return nil
	})

	go insertBlock()

	gs.RegisterService("batch-import", func(c context.Context) error {

		svc := fiber.New(fiber.Config{
			Prefork:               false,
			ServerHeader:          "Ankr team",
			DisableStartupMessage: true,
			StreamRequestBody:     true,
			BodyLimit:             500 * 1024 * 1024,
		})

		svc.Post("/blocks", blockImport)

		addr := ctx.String(utils.SvcHost.Name) + ":" + ctx.String(utils.SvcPort.Name)
		log.Info("batch import service boot", "entrypoint", addr)
		if err := svc.Listen(addr); err != nil {
			log.Error("batch import service boot", "err", err)
			return err
		}

		return nil
	})

	gs.Wait()
	return nil
}

func blockImport(ctx *fiber.Ctx) (err error) {
	var extblks = []*extblock{}
	err = ctx.BodyParser(&extblks)
	if err != nil {
		ctx.SendStatus(http.StatusInternalServerError)
		return err
	}

	if len(extblks) > 0 {
		var blks = make(types.Blocks, len(extblks))
		for i, bk := range extblks {
			blks[i] = types.NewBlockWithHeader(bk.Header).WithBody(bk.Txs, bk.Uncles).WithWithdrawals(bk.Withdrawals)
		}
		chanBlocks <- blks
	}

	return err
}

func insertBlock() {
	for {
		select {
		case blks := <-chanBlocks:
			_, err = ethereum.BlockChain().InsertChain(blks)
			if err != nil {
				os.Exit(0)
			}
		}
	}
}

type extblock struct {
	Header      *types.Header
	Txs         types.Transactions  `json:"transactions"`
	Uncles      []*types.Header     `json:"uncles"`
	Withdrawals []*types.Withdrawal `json:"withdrawals,omitempty"`
}
