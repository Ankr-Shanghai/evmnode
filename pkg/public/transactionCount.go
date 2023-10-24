package public

func GetTransactionCount(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	if params == nil {
		return nil, &ErrJson{Code: ErrCode, Message: "missing params value"}
	}

	address, blockNumber, err := addressAndNumberParams(params)
	if err != nil {
		return nil, err
	}

	count, errg := ethAPI.TxAPI.GetTransactionCount(ethAPI.Ctx, address, blockNumber)
	if errg != nil {
		return nil, &ErrJson{Code: ErrCode, Message: errg.Error()}
	}

	return count, nil
}

func GetBlockTransactionCountByHash(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	if params == nil {
		return nil, &ErrJson{Code: ErrCode, Message: "missing params value"}
	}

	hash, err := hashParams(params)
	if err != nil {
		return nil, err
	}

	return ethAPI.TxAPI.GetBlockTransactionCountByHash(ethAPI.Ctx, hash), nil

}

func GetBlockTransactionCountByNumber(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	if params == nil {
		return nil, &ErrJson{Code: ErrCode, Message: "missing params value"}
	}

	number, err := numberParams(params)
	if err != nil {
		return nil, err
	}

	return ethAPI.TxAPI.GetBlockTransactionCountByNumber(ethAPI.Ctx, number), nil
}

func GetUncleCountByBlockHash(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	return nil, &ErrJson{ErrNotExistCode, ErrNotExistMsg}
}

func GetUncleCountByBlockNumber(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	return nil, &ErrJson{ErrNotExistCode, ErrNotExistMsg}
}
