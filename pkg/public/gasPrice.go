package public

func GasPrice(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	gas, err := ethAPI.EthAPI.GasPrice(ethAPI.Ctx)
	if err != nil {
		return nil, &ErrJson{ErrCode, err.Error()}
	}
	return gas, nil
}
