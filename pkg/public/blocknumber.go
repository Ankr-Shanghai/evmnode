package public

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/log"
)

func BlockNumber(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {

	// Restore the last known head block
	head := rawdb.ReadHeadBlockHash(ethAPI.ChainDb)
	if head == (common.Hash{}) {
		// Corrupt or empty database, init from scratch
		log.Error("Empty database, shouldn't happen")
	}
	// Make sure the entire head block is available
	headBlock := ethAPI.Chain.GetBlockByHash(head)
	log.Info("BlockNumber", "rsp", headBlock.Number().String())
	return hexutil.EncodeBig(headBlock.Number()), nil
}
