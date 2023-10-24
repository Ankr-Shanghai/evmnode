package public

func GetStorageAt(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {

	if params == nil {
		return nil, &ErrJson{Code: ErrCode, Message: "missing params value"}
	}

	address, index, blockNumber, err := addressAndIndexAndNumberOrHashParams(params)
	if err != nil {
		return nil, err
	}

	storage, errg := ethAPI.BcAPI.GetStorageAt(ethAPI.Ctx, address, index, blockNumber)
	if errg != nil {
		return nil, &ErrJson{Code: ErrCode, Message: errg.Error()}
	}

	return storage, nil
}

func GetCode(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {

	if params == nil {
		return nil, &ErrJson{Code: ErrCode, Message: "missing params value"}
	}

	address, blockNumber, err := addressAndNumberParams(params)
	if err != nil {
		return nil, err
	}

	code, errg := ethAPI.BcAPI.GetCode(ethAPI.Ctx, address, blockNumber)
	if errg != nil {
		return nil, &ErrJson{Code: ErrCode, Message: errg.Error()}
	}

	return code, nil
}
