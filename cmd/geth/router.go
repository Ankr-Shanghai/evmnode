package main

import (
	"context"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/pkg/public"
	"github.com/gofiber/fiber/v2"
)

func setRouter(router fiber.Router, ethAPI *eth.EthAPIBackend) {
	router.Get("/status", status)
	router.Post("/block", blockHandler)

	api := &public.RpcAPI{
		Ctx:    context.Background(),
		BcAPI:  ethapi.NewBlockChainAPI(ethAPI),
		EthAPI: ethapi.NewEthereumAPI(ethAPI),
		TxAPI:  ethapi.NewTransactionAPI(ethAPI, new(ethapi.AddrLocker)),
	}
	router.Post("/rpc", func(ctx *fiber.Ctx) error {
		return public.RpcHandler(ctx, api)
	})
}

// status check
func status(c *fiber.Ctx) error {
	log.Info("service is ok")
	c.SendStatus(fiber.StatusOK)
	return nil
}
