package tracers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	CALL               = "call"
	CALLCODE           = "callcode"
	DELEGATECALL       = "delegatecall"
	STATICCALL         = "staticcall"
	CREATE             = "create"
	SUICIDE            = "suicide"
	REWARD             = "reward"
	TraceTypeTrace     = "trace"
	TraceTypeStateDiff = "stateDiff"
	TraceTypeVmTrace   = "vmTrace"
)

// TraceCallParam (see SendTxArgs -- this allows optional prams plus don't use MixedcaseAddress
type TraceCallParam struct {
	From                 *common.Address   `json:"from"`
	To                   *common.Address   `json:"to"`
	Gas                  *hexutil.Uint64   `json:"gas"`
	GasPrice             *hexutil.Big      `json:"gasPrice"`
	MaxPriorityFeePerGas *hexutil.Big      `json:"maxPriorityFeePerGas"`
	MaxFeePerGas         *hexutil.Big      `json:"maxFeePerGas"`
	MaxFeePerBlobGas     *hexutil.Big      `json:"maxFeePerBlobGas"`
	Value                *hexutil.Big      `json:"value"`
	Data                 hexutil.Bytes     `json:"data"`
	AccessList           *types.AccessList `json:"accessList"`
	txHash               *common.Hash
	traceTypes           []string
}

// TraceCallResult is the response to `trace_call` method
type TraceCallResult struct {
	Output          hexutil.Bytes                        `json:"output"`
	StateDiff       map[common.Address]*StateDiffAccount `json:"stateDiff"`
	Trace           []*ParityTrace                       `json:"trace"`
	VmTrace         *VmTrace                             `json:"vmTrace"`
	TransactionHash *common.Hash                         `json:"transactionHash,omitempty"`
}

// StateDiffAccount is the part of `trace_call` response that is under "stateDiff" tag
type StateDiffAccount struct {
	Balance interface{}                            `json:"balance"` // Can be either string "=" or mapping "*" => {"from": "hex", "to": "hex"}
	Code    interface{}                            `json:"code"`
	Nonce   interface{}                            `json:"nonce"`
	Storage map[common.Hash]map[string]interface{} `json:"storage"`
}

type StateDiffBalance struct {
	From *hexutil.Big `json:"from"`
	To   *hexutil.Big `json:"to"`
}

type StateDiffCode struct {
	From hexutil.Bytes `json:"from"`
	To   hexutil.Bytes `json:"to"`
}

type StateDiffNonce struct {
	From hexutil.Uint64 `json:"from"`
	To   hexutil.Uint64 `json:"to"`
}

type StateDiffStorage struct {
	From common.Hash `json:"from"`
	To   common.Hash `json:"to"`
}

// VmTrace is the part of `trace_call` response that is under "vmTrace" tag
type VmTrace struct {
	Code hexutil.Bytes `json:"code"`
	Ops  []*VmTraceOp  `json:"ops"`
}

// VmTraceOp is one element of the vmTrace ops trace
type VmTraceOp struct {
	Cost int        `json:"cost"`
	Ex   *VmTraceEx `json:"ex"`
	Pc   int        `json:"pc"`
	Sub  *VmTrace   `json:"sub"`
	Op   string     `json:"op,omitempty"`
	Idx  string     `json:"idx,omitempty"`
}

type VmTraceEx struct {
	Mem   *VmTraceMem   `json:"mem"`
	Push  []string      `json:"push"`
	Store *VmTraceStore `json:"store"`
	Used  int           `json:"used"`
}

type VmTraceMem struct {
	Data string `json:"data"`
	Off  int    `json:"off"`
}

type VmTraceStore struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

// ToMessage converts CallArgs to the Message type used by the core evm
func (args *TraceCallParam) ToMessage(statedb *state.StateDB, globalGasCap uint64, baseFee *big.Int) (core.Message, error) {
	// Set sender address or use zero address if none specified.
	var addr common.Address
	if args.From != nil {
		addr = *args.From
	}

	// Set default gas & gas price if none were set
	gas := globalGasCap
	if gas == 0 {
		gas = uint64(math.MaxUint64 / 2)
	}
	if args.Gas != nil {
		gas = uint64(*args.Gas)
	}
	if globalGasCap != 0 && globalGasCap < gas {
		log.Warn("Caller gas above allowance, capping", "requested", gas, "cap", globalGasCap)
		gas = globalGasCap
	}
	var (
		gasPrice         *uint256.Int
		gasFeeCap        *uint256.Int
		gasTipCap        *uint256.Int
		maxFeePerBlobGas *uint256.Int
	)
	if baseFee == nil {
		// If there's no basefee, then it must be a non-1559 execution
		gasPrice = new(uint256.Int)
		if args.GasPrice != nil {
			overflow := gasPrice.SetFromBig(args.GasPrice.ToInt())
			if overflow {
				return core.Message{}, fmt.Errorf("args.GasPrice higher than 2^256-1")
			}
		}
		gasFeeCap, gasTipCap = gasPrice, gasPrice
	} else {
		// A basefee is provided, necessitating 1559-type execution
		if args.GasPrice != nil {
			var overflow bool
			// User specified the legacy gas field, convert to 1559 gas typing
			gasPrice, overflow = uint256.FromBig(args.GasPrice.ToInt())
			if overflow {
				return core.Message{}, fmt.Errorf("args.GasPrice higher than 2^256-1")
			}
			gasFeeCap, gasTipCap = gasPrice, gasPrice
		} else {
			// User specified 1559 gas feilds (or none), use those
			gasFeeCap = new(uint256.Int)
			if args.MaxFeePerGas != nil {
				overflow := gasFeeCap.SetFromBig(args.MaxFeePerGas.ToInt())
				if overflow {
					return core.Message{}, fmt.Errorf("args.GasPrice higher than 2^256-1")
				}
			}
			gasTipCap = new(uint256.Int)
			if args.MaxPriorityFeePerGas != nil {
				overflow := gasTipCap.SetFromBig(args.MaxPriorityFeePerGas.ToInt())
				if overflow {
					return core.Message{}, fmt.Errorf("args.GasPrice higher than 2^256-1")
				}
			}
			// Backfill the legacy gasPrice for EVM execution, unless we're all zeroes
			gasPrice = new(uint256.Int)
			if !gasFeeCap.IsZero() || !gasTipCap.IsZero() {
				baseFeeBig, overflow := uint256.FromBig(baseFee)
				if overflow {
					return core.Message{}, fmt.Errorf("basefee higher than 2^256-1")
				}

				gasPrice = U256Min(new(uint256.Int).Add(gasTipCap, baseFeeBig), gasFeeCap)
			} else {
				// This means gasFeeCap == 0, gasTipCap == 0
				overflow := gasPrice.SetFromBig(baseFee)
				if overflow {
					return core.Message{}, fmt.Errorf("basefee higher than 2^256-1")
				}
				gasFeeCap, gasTipCap = gasPrice, gasPrice
			}
		}
		if args.MaxFeePerBlobGas != nil {
			maxFeePerBlobGas.SetFromBig(args.MaxFeePerBlobGas.ToInt())
		}
	}
	value := new(uint256.Int)
	if args.Value != nil {
		overflow := value.SetFromBig(args.Value.ToInt())
		if overflow {
			return core.Message{}, fmt.Errorf("args.Value higher than 2^256-1")
		}
	}
	var data []byte
	if args.Data != nil {
		data = args.Data
	}
	var accessList types.AccessList
	if args.AccessList != nil {
		accessList = *args.AccessList
	}

	msg := core.Message{
		From:          addr,
		To:            args.To,
		Nonce:         statedb.GetNonce(addr),
		Value:         value.ToBig(),
		GasLimit:      gas,
		GasPrice:      gasPrice.ToBig(),
		GasFeeCap:     gasFeeCap.ToBig(),
		GasTipCap:     gasTipCap.ToBig(),
		Data:          data,
		AccessList:    accessList,
		BlobGasFeeCap: maxFeePerBlobGas.ToBig(),
	}
	return msg, nil
}

// OpenEthereum-style tracer
type OeTracer struct {
	r            *TraceCallResult
	traceAddr    []int
	traceStack   []*ParityTrace
	precompile   bool // Whether the last CaptureStart was called with `precompile = true`
	compat       bool // Bug for bug compatibility mode
	lastVmOp     *VmTraceOp
	lastOp       vm.OpCode
	lastMemOff   uint64
	lastMemLen   uint64
	memOffStack  []uint64
	memLenStack  []uint64
	lastOffStack *VmTraceOp
	vmOpStack    []*VmTraceOp // Stack of vmTrace operations as call depth increases
	idx          []string     // Prefix for the "idx" inside operations, for easier navigation
	statedb      *state.StateDB
	chainRules   params.Rules
	evm          *vm.EVM
}

var _ Tracer = (*OeTracer)(nil)

func newOeTracer(statedb *state.StateDB, chainRules params.Rules) *OeTracer {
	return &OeTracer{
		r: &TraceCallResult{
			Trace:   []*ParityTrace{},
			VmTrace: &VmTrace{Ops: []*VmTraceOp{}},
		},
		traceAddr:  []int{},
		statedb:    statedb,
		chainRules: chainRules,
	}
}

func (ot *OeTracer) setEVM(evm *vm.EVM) {
	ot.evm = evm
}

func (ot *OeTracer) CaptureTxStart(gasLimit uint64) {}

func (ot *OeTracer) CaptureTxEnd(restGas uint64) {}

func (ot *OeTracer) captureStartOrEnter(deep bool, typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	_, precompile := ot.evm.Precompile(to)
	//fmt.Printf("captureStartOrEnter deep %t, typ %s, from %x, to %x, create %t, input %x, gas %d, value %d, precompile %t\n", deep, typ.String(), from, to, create, input, gas, value, precompile)
	if ot.r.VmTrace != nil {
		var vmTrace *VmTrace
		if deep {
			var vmT *VmTrace
			if len(ot.vmOpStack) > 0 {
				vmT = ot.vmOpStack[len(ot.vmOpStack)-1].Sub
			} else {
				vmT = ot.r.VmTrace
			}
			if !ot.compat {
				ot.idx = append(ot.idx, fmt.Sprintf("%d-", len(vmT.Ops)-1))
			}
		}
		if ot.lastVmOp != nil {
			vmTrace = &VmTrace{Ops: []*VmTraceOp{}}
			ot.lastVmOp.Sub = vmTrace
			ot.vmOpStack = append(ot.vmOpStack, ot.lastVmOp)
		} else {
			vmTrace = ot.r.VmTrace
		}
		if typ == vm.CREATE || typ == vm.CREATE2 {
			vmTrace.Code = common.CopyBytes(input)
			if ot.lastVmOp != nil {
				ot.lastVmOp.Cost += int(gas)
			}
		} else {
			vmTrace.Code = ot.statedb.GetCode(to)
		}
	}
	if precompile && deep && (value == nil || value.Cmp(big.NewInt(0)) == 0) {
		ot.precompile = true
		return
	}
	if gas > 500000000 {
		gas = 500000001 - (0x8000000000000000 - gas)
	}
	trace := &ParityTrace{}
	if typ == vm.CREATE {
		trResult := &CreateTraceResult{}
		trace.Type = CREATE
		trResult.Address = new(common.Address)
		copy(trResult.Address[:], to.Bytes())
		trace.Result = trResult
	} else {
		trace.Result = &TraceResult{}
		trace.Type = CALL
	}
	if deep {
		topTrace := ot.traceStack[len(ot.traceStack)-1]
		traceIdx := topTrace.Subtraces
		ot.traceAddr = append(ot.traceAddr, traceIdx)
		topTrace.Subtraces++
		if typ == vm.DELEGATECALL {
			switch action := topTrace.Action.(type) {
			case *CreateTraceAction:
				value = big.NewInt(0).Set(action.Value.ToInt())
			case *CallTraceAction:
				value = big.NewInt(0).Set(action.Value.ToInt())
			}
		}
		if typ == vm.STATICCALL {
			value = big.NewInt(0)
		}
	}
	trace.TraceAddress = make([]int, len(ot.traceAddr))
	copy(trace.TraceAddress, ot.traceAddr)
	if typ == vm.CREATE || typ == vm.CREATE2 {
		action := CreateTraceAction{}
		action.From = from
		action.Gas.ToInt().SetUint64(gas)
		action.Init = common.CopyBytes(input)
		action.Value.ToInt().Set(value)
		trace.Action = &action
	} else if typ == vm.SELFDESTRUCT {
		trace.Type = SUICIDE
		trace.Result = nil
		action := &SuicideTraceAction{}
		action.Address = from
		action.RefundAddress = to
		action.Balance.ToInt().Set(value)
		trace.Action = action
	} else {
		action := CallTraceAction{}
		switch typ {
		case vm.CALL:
			action.CallType = CALL
		case vm.CALLCODE:
			action.CallType = CALLCODE
		case vm.DELEGATECALL:
			action.CallType = DELEGATECALL
		case vm.STATICCALL:
			action.CallType = STATICCALL
		}
		action.From = from
		action.To = to
		action.Gas.ToInt().SetUint64(gas)
		action.Input = common.CopyBytes(input)
		action.Value.ToInt().Set(value)
		trace.Action = &action
	}

	ot.r.Trace = append(ot.r.Trace, trace)
	ot.traceStack = append(ot.traceStack, trace)
}

func (ot *OeTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	ot.captureStartOrEnter(false /* deep */, vm.CALL, from, to, input, gas, value)
}

func (ot *OeTracer) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	ot.captureStartOrEnter(true /* deep */, typ, from, to, input, gas, value)
}

func (ot *OeTracer) captureEndOrExit(deep bool, output []byte, usedGas uint64, err error) {
	if ot.r.VmTrace != nil {
		if len(ot.vmOpStack) > 0 {
			ot.lastOffStack = ot.vmOpStack[len(ot.vmOpStack)-1]
			ot.vmOpStack = ot.vmOpStack[:len(ot.vmOpStack)-1]
		}
		if !ot.compat && deep {
			ot.idx = ot.idx[:len(ot.idx)-1]
		}
		if deep {
			ot.lastMemOff = ot.memOffStack[len(ot.memOffStack)-1]
			ot.memOffStack = ot.memOffStack[:len(ot.memOffStack)-1]
			ot.lastMemLen = ot.memLenStack[len(ot.memLenStack)-1]
			ot.memLenStack = ot.memLenStack[:len(ot.memLenStack)-1]
		}
	}
	if ot.precompile {
		ot.precompile = false
		return
	}
	if !deep {
		ot.r.Output = common.CopyBytes(output)
	}
	ignoreError := false
	topTrace := ot.traceStack[len(ot.traceStack)-1]
	if ot.compat {
		ignoreError = !deep && topTrace.Type == CREATE
	}
	if err != nil && !ignoreError {
		if err == vm.ErrExecutionReverted {
			topTrace.Error = "Reverted"
			switch topTrace.Type {
			case CALL:
				topTrace.Result.(*TraceResult).GasUsed = new(hexutil.Big)
				topTrace.Result.(*TraceResult).GasUsed.ToInt().SetUint64(usedGas)
				topTrace.Result.(*TraceResult).Output = common.CopyBytes(output)
			case CREATE:
				topTrace.Result.(*CreateTraceResult).GasUsed = new(hexutil.Big)
				topTrace.Result.(*CreateTraceResult).GasUsed.ToInt().SetUint64(usedGas)
				topTrace.Result.(*CreateTraceResult).Code = common.CopyBytes(output)
			}
		} else {
			topTrace.Result = nil
			topTrace.Error = err.Error()
		}
	} else {
		if len(output) > 0 {
			switch topTrace.Type {
			case CALL:
				topTrace.Result.(*TraceResult).Output = common.CopyBytes(output)
			case CREATE:
				topTrace.Result.(*CreateTraceResult).Code = common.CopyBytes(output)
			}
		}
		switch topTrace.Type {
		case CALL:
			topTrace.Result.(*TraceResult).GasUsed = new(hexutil.Big)
			topTrace.Result.(*TraceResult).GasUsed.ToInt().SetUint64(usedGas)
		case CREATE:
			topTrace.Result.(*CreateTraceResult).GasUsed = new(hexutil.Big)
			topTrace.Result.(*CreateTraceResult).GasUsed.ToInt().SetUint64(usedGas)
		}
	}
	ot.traceStack = ot.traceStack[:len(ot.traceStack)-1]
	if deep {
		ot.traceAddr = ot.traceAddr[:len(ot.traceAddr)-1]
	}
}

func (ot *OeTracer) CaptureEnd(output []byte, usedGas uint64, err error) {
	ot.captureEndOrExit(false /* deep */, output, usedGas, err)
}

func (ot *OeTracer) CaptureExit(output []byte, usedGas uint64, err error) {
	ot.captureEndOrExit(true /* deep */, output, usedGas, err)
}

func (ot *OeTracer) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, opDepth int, err error) {
	memory := scope.Memory
	st := scope.Stack

	if ot.r.VmTrace != nil {
		var vmTrace *VmTrace
		if len(ot.vmOpStack) > 0 {
			vmTrace = ot.vmOpStack[len(ot.vmOpStack)-1].Sub
		} else {
			vmTrace = ot.r.VmTrace
		}
		if ot.lastVmOp != nil && ot.lastVmOp.Ex != nil {
			// Set the "push" of the last operation
			var showStack int
			switch {
			case ot.lastOp >= vm.PUSH0 && ot.lastOp <= vm.PUSH32:
				showStack = 1
			case ot.lastOp >= vm.SWAP1 && ot.lastOp <= vm.SWAP16:
				showStack = int(ot.lastOp-vm.SWAP1) + 2
			case ot.lastOp >= vm.DUP1 && ot.lastOp <= vm.DUP16:
				showStack = int(ot.lastOp-vm.DUP1) + 2
			}
			switch ot.lastOp {
			case vm.CALLDATALOAD, vm.SLOAD, vm.MLOAD, vm.CALLDATASIZE, vm.LT, vm.GT, vm.DIV, vm.SDIV, vm.SAR, vm.AND, vm.EQ, vm.CALLVALUE, vm.ISZERO,
				vm.ADD, vm.EXP, vm.CALLER, vm.KECCAK256, vm.SUB, vm.ADDRESS, vm.GAS, vm.MUL, vm.RETURNDATASIZE, vm.NOT, vm.SHR, vm.SHL,
				vm.EXTCODESIZE, vm.SLT, vm.OR, vm.NUMBER, vm.PC, vm.TIMESTAMP, vm.BALANCE, vm.SELFBALANCE, vm.MULMOD, vm.ADDMOD, vm.BASEFEE,
				vm.BLOCKHASH, vm.BYTE, vm.XOR, vm.ORIGIN, vm.CODESIZE, vm.MOD, vm.SIGNEXTEND, vm.GASLIMIT, vm.DIFFICULTY, vm.SGT, vm.GASPRICE,
				vm.MSIZE, vm.EXTCODEHASH, vm.SMOD, vm.CHAINID, vm.COINBASE:
				showStack = 1
			}
			for i := showStack - 1; i >= 0; i-- {
				if st.Len() > i {
					ot.lastVmOp.Ex.Push = append(ot.lastVmOp.Ex.Push, st.Back(i).String())
				}
			}
			// Set the "mem" of the last operation
			var setMem bool
			switch ot.lastOp {
			case vm.MSTORE, vm.MSTORE8, vm.MLOAD, vm.RETURNDATACOPY, vm.CALLDATACOPY, vm.CODECOPY, vm.EXTCODECOPY:
				setMem = true
			}
			if setMem && ot.lastMemLen > 0 {
				cpy := memory.GetCopy(int64(ot.lastMemOff), int64(ot.lastMemLen))
				if len(cpy) == 0 {
					cpy = make([]byte, ot.lastMemLen)
				}
				ot.lastVmOp.Ex.Mem = &VmTraceMem{Data: fmt.Sprintf("0x%0x", cpy), Off: int(ot.lastMemOff)}
			}
		}
		if ot.lastOffStack != nil {
			ot.lastOffStack.Ex.Used = int(gas)
			if st.Len() > 0 {
				ot.lastOffStack.Ex.Push = []string{st.Back(0).String()}
			} else {
				ot.lastOffStack.Ex.Push = []string{}
			}
			if ot.lastMemLen > 0 && memory != nil {
				cpy := memory.GetCopy(int64(ot.lastMemOff), int64(ot.lastMemLen))
				if len(cpy) == 0 {
					cpy = make([]byte, ot.lastMemLen)
				}
				ot.lastOffStack.Ex.Mem = &VmTraceMem{Data: fmt.Sprintf("0x%0x", cpy), Off: int(ot.lastMemOff)}
			}
			ot.lastOffStack = nil
		}
		if ot.lastOp == vm.STOP && op == vm.STOP && len(ot.vmOpStack) == 0 {
			// Looks like OE is "optimising away" the second STOP
			return
		}
		ot.lastVmOp = &VmTraceOp{Ex: &VmTraceEx{}}
		vmTrace.Ops = append(vmTrace.Ops, ot.lastVmOp)
		if !ot.compat {
			var sb strings.Builder
			sb.Grow(len(ot.idx))
			for _, idx := range ot.idx {
				sb.WriteString(idx)
			}
			ot.lastVmOp.Idx = fmt.Sprintf("%s%d", sb.String(), len(vmTrace.Ops)-1)
		}
		ot.lastOp = op
		ot.lastVmOp.Cost = int(cost)
		ot.lastVmOp.Pc = int(pc)
		ot.lastVmOp.Ex.Push = []string{}
		ot.lastVmOp.Ex.Used = int(gas) - int(cost)
		if !ot.compat {
			ot.lastVmOp.Op = op.String()
		}
		switch op {
		case vm.MSTORE, vm.MLOAD:
			if st.Len() > 0 {
				ot.lastMemOff = st.Back(0).Uint64()
				ot.lastMemLen = 32
			}
		case vm.MSTORE8:
			if st.Len() > 0 {
				ot.lastMemOff = st.Back(0).Uint64()
				ot.lastMemLen = 1
			}
		case vm.RETURNDATACOPY, vm.CALLDATACOPY, vm.CODECOPY:
			if st.Len() > 2 {
				ot.lastMemOff = st.Back(0).Uint64()
				ot.lastMemLen = st.Back(2).Uint64()
			}
		case vm.EXTCODECOPY:
			if st.Len() > 3 {
				ot.lastMemOff = st.Back(1).Uint64()
				ot.lastMemLen = st.Back(3).Uint64()
			}
		case vm.STATICCALL, vm.DELEGATECALL:
			if st.Len() > 5 {
				ot.memOffStack = append(ot.memOffStack, st.Back(4).Uint64())
				ot.memLenStack = append(ot.memLenStack, st.Back(5).Uint64())
			}
		case vm.CALL, vm.CALLCODE:
			if st.Len() > 6 {
				ot.memOffStack = append(ot.memOffStack, st.Back(5).Uint64())
				ot.memLenStack = append(ot.memLenStack, st.Back(6).Uint64())
			}
		case vm.CREATE, vm.CREATE2, vm.SELFDESTRUCT:
			// Effectively disable memory output
			ot.memOffStack = append(ot.memOffStack, 0)
			ot.memLenStack = append(ot.memLenStack, 0)
		case vm.SSTORE:
			if st.Len() > 1 {
				ot.lastVmOp.Ex.Store = &VmTraceStore{Key: st.Back(0).String(), Val: st.Back(1).String()}
			}
		}
		if ot.lastVmOp.Ex.Used < 0 {
			ot.lastVmOp.Ex = nil
		}
	}
}

func (ot *OeTracer) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, opDepth int, err error) {
}

func (*OeTracer) CaptureSystemTxEnd(intrinsicGas uint64) {}

// GetResult returns an empty json object.
func (t *OeTracer) GetResult() (json.RawMessage, error) {
	return json.RawMessage(`{}`), nil
}

// GetResult returns an empty json object.
func (t *OeTracer) getTraceResult() (*TraceCallResult, error) {
	return t.r, nil
}

// Stop terminates execution of the tracer at the first opportune moment.
func (t *OeTracer) Stop(err error) {
}

// Implements core/state/StateWriter to provide state diffs
type StateDiff struct {
	evm   *vm.EVM
	sdMap map[common.Address]*StateDiffAccount
}

func (sd *StateDiff) UpdateAccountData(address common.Address) error {
	if _, ok := sd.evm.Precompile(address); ok {
		return nil
	}
	if _, ok := sd.sdMap[address]; !ok {
		sd.sdMap[address] = &StateDiffAccount{Storage: make(map[common.Hash]map[string]interface{})}
	}
	return nil
}

func (sd *StateDiff) UpdateAccountCode(address common.Address) error {
	if _, ok := sd.evm.Precompile(address); ok {
		return nil
	}
	if _, ok := sd.sdMap[address]; !ok {
		sd.sdMap[address] = &StateDiffAccount{Storage: make(map[common.Hash]map[string]interface{})}
	}
	return nil
}

func (sd *StateDiff) DeleteAccount(address common.Address) error {
	if _, ok := sd.evm.Precompile(address); ok {
		return nil
	}
	if _, ok := sd.sdMap[address]; !ok {
		sd.sdMap[address] = &StateDiffAccount{Storage: make(map[common.Hash]map[string]interface{})}
	}
	return nil
}

func (sd *StateDiff) WriteAccountStorage(address common.Address, key common.Hash, original, value *big.Int) error {
	if _, ok := sd.evm.Precompile(address); ok {
		return nil
	}
	if original.Cmp(value) == 0 {
		return nil
	}
	accountDiff := sd.sdMap[address]
	if accountDiff == nil {
		accountDiff = &StateDiffAccount{Storage: make(map[common.Hash]map[string]interface{})}
		sd.sdMap[address] = accountDiff
	}
	m := make(map[string]interface{})
	m["*"] = &StateDiffStorage{From: common.BytesToHash(original.Bytes()), To: common.BytesToHash(value.Bytes())}
	accountDiff.Storage[key] = m
	return nil
}

func (sd *StateDiff) CreateContract(address common.Address) error {
	if _, ok := sd.evm.Precompile(address); ok {
		return nil
	}
	if _, ok := sd.sdMap[address]; !ok {
		sd.sdMap[address] = &StateDiffAccount{Storage: make(map[common.Hash]map[string]interface{})}
	}
	return nil
}

// CompareStates uses the addresses accumulated in the sdMap and compares balances, nonces, and codes of the accounts, and fills the rest of the sdMap
func (sd *StateDiff) CompareStates(initialIbs, ibs *state.StateDB) {
	var toRemove []common.Address
	for addr, accountDiff := range sd.sdMap {
		initialExist := initialIbs.Exist(addr)
		exist := ibs.Exist(addr)
		if initialExist {
			if exist {
				var allEqual = len(accountDiff.Storage) == 0
				fromBalance := initialIbs.GetBalance(addr)
				toBalance := ibs.GetBalance(addr)
				if fromBalance.Cmp(toBalance) == 0 {
					accountDiff.Balance = "="
				} else {
					m := make(map[string]*StateDiffBalance)
					m["*"] = &StateDiffBalance{From: (*hexutil.Big)(fromBalance), To: (*hexutil.Big)(toBalance)}
					accountDiff.Balance = m
					allEqual = false
				}
				fromCode := initialIbs.GetCode(addr)
				toCode := ibs.GetCode(addr)
				if bytes.Equal(fromCode, toCode) {
					accountDiff.Code = "="
				} else {
					m := make(map[string]*StateDiffCode)
					m["*"] = &StateDiffCode{From: fromCode, To: toCode}
					accountDiff.Code = m
					allEqual = false
				}
				fromNonce := initialIbs.GetNonce(addr)
				toNonce := ibs.GetNonce(addr)
				if fromNonce == toNonce {
					accountDiff.Nonce = "="
				} else {
					m := make(map[string]*StateDiffNonce)
					m["*"] = &StateDiffNonce{From: hexutil.Uint64(fromNonce), To: hexutil.Uint64(toNonce)}
					accountDiff.Nonce = m
					allEqual = false
				}
				if allEqual {
					toRemove = append(toRemove, addr)
				}
			} else {
				{
					m := make(map[string]*hexutil.Big)
					m["-"] = (*hexutil.Big)(initialIbs.GetBalance(addr))
					accountDiff.Balance = m
				}
				{
					m := make(map[string]hexutil.Bytes)
					m["-"] = initialIbs.GetCode(addr)
					accountDiff.Code = m
				}
				{
					m := make(map[string]hexutil.Uint64)
					m["-"] = hexutil.Uint64(initialIbs.GetNonce(addr))
					accountDiff.Nonce = m
				}
			}
		} else if exist {
			{
				m := make(map[string]*hexutil.Big)
				m["+"] = (*hexutil.Big)(ibs.GetBalance(addr))
				accountDiff.Balance = m
			}
			{
				m := make(map[string]hexutil.Bytes)
				m["+"] = ibs.GetCode(addr)
				accountDiff.Code = m
			}
			{
				m := make(map[string]hexutil.Uint64)
				m["+"] = hexutil.Uint64(ibs.GetNonce(addr))
				accountDiff.Nonce = m
			}
			// Transform storage
			for _, sm := range accountDiff.Storage {
				str := sm["*"].(*StateDiffStorage)
				delete(sm, "*")
				sm["+"] = &str.To
			}
		} else {
			toRemove = append(toRemove, addr)
		}
	}
	for _, addr := range toRemove {
		delete(sd.sdMap, addr)
	}
}

// U256Min returns the smaller of x or y.
func U256Min(x, y *uint256.Int) *uint256.Int {
	if x.Cmp(y) > 0 {
		return y
	}
	return x
}
