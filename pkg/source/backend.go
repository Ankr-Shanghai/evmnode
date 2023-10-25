package source

import (
	"os"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/urfave/cli/v2"
)

var (
	BackendClient *ethclient.Client
	err           error
)

func InitBackendClient(ctx *cli.Context) {
	burl := ctx.String("backend")

	BackendClient, err = ethclient.Dial(burl)

	if err != nil {
		log.Error("ethclient.Dial", "err", err)
		os.Exit(-1)
	}
}
