package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/rpc"
)

func getAllAPIs() []rpc.API {
	apis := ethereum.APIs()
	apis = append(apis, tracers.APIs(ethereum.APIBackend)...)
	apis = append(apis, extapis()...)

	return apis
}

func extapis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "web3",
			Service:   &web3API{},
		}, {
			Namespace: "net",
			Service:   newNetAPI(ethereum.NetworkID),
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

type netAPI struct {
	networkVersion uint64
}

// NewNetAPI creates a new net API instance.
func newNetAPI(networkVersion uint64) *netAPI {
	return &netAPI{networkVersion}
}

// Listening returns an indication if the node is listening for network connections.
func (s *netAPI) Listening() bool {
	return true // always listening
}

// PeerCount returns the number of connected peers
func (s *netAPI) PeerCount() hexutil.Uint {
	return hexutil.Uint(0)
}

// Version returns the current ethereum protocol version.
func (s *netAPI) Version() string {
	return fmt.Sprintf("%d", s.networkVersion)
}
