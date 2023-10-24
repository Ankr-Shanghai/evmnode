package public

import (
	"github.com/ethereum/go-ethereum/rpc"
)

func GetBlockByNumber(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {

	if params == nil {
		return nil, &ErrJson{Code: ErrCode, Message: "missing params value"}
	}
	number, fullTx, err := gbnParams(params)
	if err != nil {
		return nil, err
	}

	blk, errg := ethAPI.BcAPI.GetBlockByNumber(ethAPI.Ctx, number, fullTx)
	if errg != nil {
		return nil, &ErrJson{Code: ErrCode, Message: errg.Error()}
	}

	return blk, nil

}

func gbnParams(params interface{}) (rpc.BlockNumber, bool, *ErrJson) {
	var number rpc.BlockNumber

	pa, ok := params.([]interface{})
	if !ok {
		return number, false, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	if len(pa) != 2 {
		return number, false, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	data, ok := pa[0].(string)
	if !ok {
		return number, false, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	errg := number.UnmarshalJSON([]byte(data))
	if errg != nil {
		return number, false, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	fullTx, ok := pa[1].(bool)
	if !ok {
		return number, false, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	return number, fullTx, nil
}
