package public

func SendTransaction(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	return nil, &ErrJson{ErrNotExistCode, ErrNotExistMsg}
}

func SendRawTransaction(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	return nil, &ErrJson{ErrNotExistCode, ErrNotExistMsg}
}
