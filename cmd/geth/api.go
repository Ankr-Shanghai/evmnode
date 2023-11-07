package main

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
)

func getAllAPIs(apiBackend *eth.EthAPIBackend) []rpc.API {
	apis := ethapi.GetAPIs(apiBackend)
	apis = append(apis, tracers.APIs(apiBackend)...)
	apis = append(apis, extapis()...)

	return apis
}

func extapis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "debug",
			Service:   debug.Handler,
		}, {
			Namespace: "web3",
			Service:   &web3API{},
		},
	}
}

type web3API struct{}

// Sha3 applies the ethereum sha3 implementation on the input.
// It assumes the input is hex encoded.
func (s *web3API) Sha3(input hexutil.Bytes) hexutil.Bytes {
	return crypto.Keccak256(input)
}

// ClientVersion returns the node name
func (s *web3API) ClientVersion() string {
	return clientVersion()
}
