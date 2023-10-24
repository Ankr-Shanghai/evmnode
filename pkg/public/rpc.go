package public

import (
	"context"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gofiber/fiber/v2"
)

type RpcAPI struct {
	Ctx     context.Context
	BcAPI   *ethapi.BlockChainAPI
	EthAPI  *ethapi.EthereumAPI
	TxAPI   *ethapi.TransactionAPI
	ChainDb ethdb.Database
	Chain   *core.BlockChain
}

type ReqJson struct {
	ID      interface{} `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Version string      `json:"jsonrpc"`
}

type RspJson struct {
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *ErrJson    `json:"error,omitempty"`
	Version string      `json:"jsonrpc"`
}

func RpcHandler(ctx *fiber.Ctx, ethAPI *RpcAPI) error {
	var (
		req ReqJson
		rsp RspJson
	)
	log.Info("rpc request: ", "body", string(ctx.Body()))

	err := ctx.BodyParser(&req)
	if err != nil {
		ctx.SendStatus(fiber.StatusBadRequest)
		return err
	}
	rsp.ID = req.ID
	rsp.Version = req.Version

	rs, errj := handlerMap[req.Method](ethAPI, req.Params)
	if errj != nil {
		rsp.Error = errj
	} else {
		rsp.Result = rs
	}

	ctx.JSON(rsp)
	return nil
}

var (
	handlerMap = map[string]func(*RpcAPI, interface{}) (interface{}, *ErrJson){
		"eth_chainId":                          ChainID,
		"eth_mining":                           Mining,
		"eth_hashrate":                         HashRate,
		"eth_syncing":                          Syncing,
		"eth_gasPrice":                         GasPrice,
		"eth_accounts":                         Accounts,
		"eth_blockNumber":                      BlockNumber,
		"eth_getBalance":                       GetBalance,
		"eth_getStorageAt":                     GetStorageAt,
		"eth_getTransactionCount":              GetTransactionCount,
		"eth_getBlockTransactionCountByHash":   GetBlockTransactionCountByHash,
		"eth_getBlockTransactionCountByNumber": GetBlockTransactionCountByNumber,
		"eth_getUncleCountByBlockHash":         GetUncleCountByBlockHash,
		"eth_getUncleCountByBlockNumber":       GetUncleCountByBlockNumber,
		"eth_getCode":                          GetCode,
		"eth_sign":                             Sign,
		"eth_signTransaction":                  SignTransaction,
		"eth_sendTransaction":                  SendTransaction,
		"eth_sendRawTransaction":               SendRawTransaction,
		"eth_call":                             Call,
		"eth_getBlockByNumber":                 GetBlockByNumber,
	}
)
