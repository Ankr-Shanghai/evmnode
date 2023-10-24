package public

func Syncing(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	return false, nil
}

func ChainID(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	return ethAPI.BcAPI.ChainId(), nil
}

func Mining(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	return nil, &ErrJson{ErrNotExistCode, ErrNotExistMsg}
}

func HashRate(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	return nil, &ErrJson{ErrNotExistCode, ErrNotExistMsg}
}

func Accounts(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	return nil, &ErrJson{ErrNotExistCode, ErrNotExistMsg}
}
func Sign(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	return nil, &ErrJson{ErrNotExistCode, ErrNotExistMsg}
}
