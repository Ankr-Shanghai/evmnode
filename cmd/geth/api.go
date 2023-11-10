package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
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
		}, {
			Namespace: "net",
			Service:   newNetAPI(ethereum.NetworkID),
		}, {
			Namespace: "eth",
			Service:   newEthereumAPI(),
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

type ethereumAPI struct {
}

// NewEthereumAccountAPI creates a new EthereumAccountAPI.
func newEthereumAPI() *ethereumAPI {
	return &ethereumAPI{}
}

// Accounts returns the collection of accounts this node manages.
func (s *ethereumAPI) Accounts() []common.Address {
	return []common.Address{}
}

func (s *ethereumAPI) Mining() bool {
	return false
}

// Etherbase is the address that mining rewards will be sent to.
func (api *ethereumAPI) Etherbase() (common.Address, error) {
	return common.Address{}, nil
}

// Coinbase is the address that mining rewards will be sent to (alias for Etherbase).
func (api *ethereumAPI) Coinbase() (common.Address, error) {
	return common.Address{}, nil
}

// Hashrate returns the POW hashrate.
func (api *ethereumAPI) Hashrate() hexutil.Uint64 {
	return hexutil.Uint64(0)
}

func (s *ethereumAPI) Syncing() bool {
	return false
}
