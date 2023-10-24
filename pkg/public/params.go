package public

import (
	"reflect"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
)

func numberParams(params interface{}) (rpc.BlockNumber, *ErrJson) {
	var number rpc.BlockNumber

	pa, ok := params.([]interface{})
	if !ok {
		return number, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	if len(pa) != 1 {
		return number, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	data, ok := pa[0].(string)
	if !ok {
		return number, &ErrJson{Code: ErrCode, Message: "wrong params block value"}
	}

	num, err := hexutil.DecodeUint64(data)
	if err != nil {
		return number, &ErrJson{Code: ErrCode, Message: "wrong params block value"}
	}
	number = rpc.BlockNumber(num)

	return number, nil
}

func hashParams(params interface{}) (common.Hash, *ErrJson) {
	var hash common.Hash

	pa, ok := params.([]interface{})
	if !ok {
		return hash, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	if len(pa) != 1 {
		return hash, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	data, ok := pa[0].(string)
	if !ok {
		return hash, &ErrJson{Code: ErrCode, Message: "wrong params hash value"}
	}

	hash = common.HexToHash(data)

	return hash, nil
}

func addressAndIndexAndNumberOrHashParams(params interface{}) (common.Address, string, rpc.BlockNumberOrHash, *ErrJson) {

	var (
		address      common.Address
		index        string
		numberOrHash rpc.BlockNumberOrHash
	)

	pa, ok := params.([]interface{})
	if !ok {
		return address, index, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	if len(pa) != 3 {
		return address, index, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	addrs, ok := pa[0].(string)
	if !ok {
		return address, index, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}
	address = common.HexToAddress(addrs)

	index, ok = pa[1].(string)
	if !ok {
		return address, index, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	data, ok := pa[1].(string)
	if !ok {
		return address, index, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params block/hash value"}
	}

	if len(data) == 66 {
		blkHash := common.HexToHash(data)
		numberOrHash = rpc.BlockNumberOrHashWithHash(blkHash, false)
	} else {
		num, err := hexutil.DecodeUint64(data)
		if err != nil {
			return address, index, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params block/hash value"}
		}
		numberOrHash = rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(num))
	}

	return address, index, numberOrHash, nil
}

func addressAndNumberParams(params interface{}) (common.Address, rpc.BlockNumberOrHash, *ErrJson) {
	var (
		address      common.Address
		numberOrHash rpc.BlockNumberOrHash
	)

	pa, ok := params.([]interface{})
	if !ok {
		return address, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	if len(pa) != 2 {
		return address, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	addrs, ok := pa[0].(string)
	if !ok {
		return address, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params address value"}
	}
	address = common.HexToAddress(addrs)

	data, ok := pa[1].(string)
	if !ok {
		return address, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params block/hash value"}
	}

	if len(data) == 66 {
		blkHash := common.HexToHash(data)
		numberOrHash = rpc.BlockNumberOrHashWithHash(blkHash, false)
	} else {
		num, err := hexutil.DecodeUint64(data)
		if err != nil {
			return address, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params block/hash value"}
		}
		numberOrHash = rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(num))
	}

	return address, numberOrHash, nil
}

func callParams(params interface{}) (ethapi.TransactionArgs, rpc.BlockNumberOrHash, *ErrJson) {
	var (
		txargs       ethapi.TransactionArgs
		numberOrHash rpc.BlockNumberOrHash
	)
	pa, ok := params.([]interface{})
	if !ok {
		return txargs, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	if len(pa) != 2 {
		return txargs, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}

	// txargs
	txface, ok := pa[0].(map[string]interface{})
	if !ok {
		return txargs, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params value"}
	}
	txt := reflect.TypeOf(txargs)
	for i := 0; i < txt.NumField(); i++ {
		field := txt.Field(i)
		switch field.Tag.Get("json") {
		case "from":
			from, ok := txface["from"].(string)
			if !ok {
				continue
			}
			*txargs.From = common.HexToAddress(from)
		case "to":
			to, ok := txface["to"].(string)
			if !ok {
				continue
			}
			*txargs.To = common.HexToAddress(to)
		case "gas":
			gas, ok := txface["gas"].(string)
			if !ok {
				continue
			}
			gast, err := hexutil.DecodeUint64(gas)
			if err != nil {
				return txargs, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params gas value"}
			}
			*txargs.Gas = hexutil.Uint64(gast)
		case "gasPrice":
			gasPrice, ok := txface["gasPrice"].(string)
			if !ok {
				continue
			}
			txgas, err := hexutil.DecodeBig(gasPrice)
			if err != nil {
				return txargs, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params gasPrice value"}
			}
			*txargs.GasPrice = hexutil.Big(*txgas)
		case "value":
		case "nonce":
		case "data":
		case "input":
		case "maxFeePerGas":
		case "maxPriorityFeePerGas":
		case "accessList":
		case "chainId":
		}
	}

	// number or hash
	data, ok := pa[1].(string)
	if !ok {
		return txargs, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params block/hash value"}
	}

	if len(data) == 66 {
		blkHash := common.HexToHash(data)
		numberOrHash = rpc.BlockNumberOrHashWithHash(blkHash, false)
	} else {
		num, err := hexutil.DecodeUint64(data)
		if err != nil {
			return txargs, numberOrHash, &ErrJson{Code: ErrCode, Message: "wrong params block/hash value"}
		}
		numberOrHash = rpc.BlockNumberOrHashWithNumber(rpc.BlockNumber(num))
	}

	return txargs, numberOrHash, nil
}
