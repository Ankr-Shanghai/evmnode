package public

import (
	"github.com/ethereum/go-ethereum/log"
)

func BlockNumber(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	number := ethAPI.BcAPI.BlockNumber()
	log.Info("BlockNumber", "rsp", number)
	return number, nil
}
