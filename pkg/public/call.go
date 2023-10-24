package public

import "fmt"

func Call(ethAPI *RpcAPI, params interface{}) (interface{}, *ErrJson) {
	if params == nil {
		return nil, &ErrJson{Code: ErrCode, Message: "missing params value"}
	}

	txargs, numberOrHash, err := callParams(params)
	if err != nil {
		return nil, err
	}

	fmt.Printf("txargs: %+v\n", txargs)
	fmt.Printf("numberOrHash: %+v\n", numberOrHash)

	return nil, nil
}
