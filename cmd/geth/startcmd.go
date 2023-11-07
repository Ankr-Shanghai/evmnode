package main

import (
	"context"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/cmd/geth/utils"
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

	gs.RegisterService("import", func(c context.Context) error {
		// make up for missing blocks
		// 1. get local latest block
		// 2. get remote latest block
		// 3. get missing blocks
		// 4. import missing blocks
		header := ethereum.BlockChain().CurrentHeader()
		remoteBlockNumber, err := source.BackendClient.BlockNumber(c)
		if err != nil {
			log.Error("import", "take remote latest block", err)
			return err
		}
		log.Info("import", "localBlockNumber", header.Number, "remoteBlockNumber", remoteBlockNumber)

		// take blocks
		chanBlocks := make(chan types.Blocks, 1024)
		wg := sync.WaitGroup{}
		wg.Add(1)
		go takeBlocks(&wg, source.BackendClient, header.Number.Uint64()+1, remoteBlockNumber, chanBlocks)
		// consume blocks
		wg.Add(1)
		go consumeBlocks(&wg, chanBlocks)
		wg.Wait()

		tickSecond := time.Tick(time.Second)
		var gerr error
		for {
			select {
			case <-tickSecond:
				header := ethereum.BlockChain().CurrentHeader()
				remoteBlockNumber, err := source.BackendClient.BlockNumber(c)
				if err != nil {
					log.Error("import", "take remote latest block", err)
					continue
				}
				log.Info("import", "localBlockNumber", header.Number, "remoteBlockNumber", remoteBlockNumber)
				for i := header.Number.Uint64() + 1; i <= remoteBlockNumber; i++ {
				DoGgainLoop:
					block, err := source.BackendClient.BlockByNumber(c, big.NewInt(int64(i)))
					if err != nil {
						gerr = err
					}
					block24, err := source.BackendClient.BlockByNumber(c, big.NewInt(int64(i-24)))
					gerr = err

					if gerr != nil {
						log.Error("import", "take remote block", err)
						time.Sleep(time.Millisecond * 100)
						goto DoGgainLoop
					}

					ethereum.BlockChain().InsertChain([]*types.Block{block, block24})
				}
			}
		}
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

		gs.RegisterService("hello", func(ctx context.Context) error {
			return nil
		})

		return nil
	})

	gs.Wait()
	return nil
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
