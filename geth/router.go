package main

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/gofiber/fiber/v2"
)

func setRouter(router fiber.Router) {
	router.Get("/status", status)
	router.Post("/block", blockHandler)
}

// status check
func status(c *fiber.Ctx) error {
	log.Info("service is ok")
	c.SendStatus(fiber.StatusOK)
	return nil
}
