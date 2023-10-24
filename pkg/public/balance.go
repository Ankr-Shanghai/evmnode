package public

import (
	"github.com/ethereum/go-ethereum/log"
)

func GetBalance(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	if params == nil {
		return nil, &ErrJson{Code: ErrCode, Message: "missing params value"}
	}
	addr, numberOrHash, err := addressAndNumberParams(params)
	if err != nil {
		return nil, err
	}

	balance, errg := ethAPI.BcAPI.GetBalance(ethAPI.Ctx, addr, numberOrHash)
	if errg != nil {
		return nil, &ErrJson{Code: ErrCode, Message: errg.Error()}
	}

	log.Info("GetBalance", "Address", addr, "Balance", balance)

	return balance, nil
}
