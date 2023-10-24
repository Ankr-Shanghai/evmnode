package main

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sunvim/utils/grace"
	"github.com/urfave/cli/v2"
)

func start(ctx *cli.Context) error {
	_, gs := grace.New(context.Background())

	// create blockchain
	newBlockChain(ctx)

	gs.Register(func() error {
		ethereum.Stop()
		return nil
	})

	gs.RegisterService("block handler", func(c context.Context) error {

		header := ethereum.BlockChain().CurrentHeader()

		log.Info("blockchain.CurrentHeader", "number", header.Number.String())

		rpcurl := "https://rpc.ankr.com/bsc"
		client, err := ethclient.Dial(rpcurl)
		if err != nil {
			log.Error("ethclient.Dial", "err", err)
			return err
		}

		var begin = header.Number.Int64()

		for i := begin + 1; i < begin+40000; i++ {
			block, err := client.BlockByNumber(c, big.NewInt(i))
			if err != nil {
				log.Error("client.BlockByNumber", "err", err)
				return err
			}
			_, err = ethereum.BlockChain().InsertChain([]*types.Block{block})
			if err != nil {
				log.Error("blockchain.InsertChain", "err", err)
				return err
			}
		}

		return nil
	})

	gs.RegisterService("evm", func(c context.Context) error {
		svc := fiber.New(fiber.Config{
			Prefork:               false,
			ServerHeader:          "Ankr team",
			DisableStartupMessage: true,
		})

		dataRouter := svc.Group("v1")
		dataRouter.Use(recover.New())
		setRouter(dataRouter, ethereum.APIBackend)

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
