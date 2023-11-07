package main

import (
	"context"

	"github.com/ethereum/go-ethereum/cmd/geth/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/pkg/source"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sunvim/utils/grace"
	"github.com/urfave/cli/v2"
)

func rpcstart(ctx *cli.Context) error {
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
		})

		svc.Use(recover.New())
		svc.Post("/", handler)

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
