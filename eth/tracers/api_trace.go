package tracers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/rpc"
)

// type TraceAPI interface {
// 	ReplayBlockTransactions(ctx context.Context, blockNr rpc.BlockNumberOrHash, traceTypes []string, gasBailOut *bool) ([]*TraceCallResult, error)
// 	ReplayTransaction(ctx context.Context, txHash libcommon.Hash, traceTypes []string, gasBailOut *bool) (*TraceCallResult, error)
// 	Call(ctx context.Context, call TraceCallParam, types []string, blockNr *rpc.BlockNumberOrHash) (*TraceCallResult, error)
// 	CallMany(ctx context.Context, calls json.RawMessage, blockNr *rpc.BlockNumberOrHash) ([]*TraceCallResult, error)
// 	RawTransaction(ctx context.Context, txHash libcommon.Hash, traceTypes []string) ([]interface{}, error)

// 	Transaction(ctx context.Context, txHash libcommon.Hash, gasBailOut *bool) (ParityTraces, error)
// 	Get(ctx context.Context, txHash libcommon.Hash, txIndicies []hexutil.Uint64, gasBailOut *bool) (*ParityTrace, error)
// 	Block(ctx context.Context, blockNr rpc.BlockNumber, gasBailOut *bool) (ParityTraces, error)
// 	Filter(ctx context.Context, req TraceFilterRequest, gasBailOut *bool, stream *jsoniter.Stream) error
// }

type APITrace struct {
	backend Backend
}

func NewTrace(backend Backend) *APITrace {
	return &APITrace{
		backend: backend,
	}
}

func (api *APITrace) Block(ctx context.Context, number rpc.BlockNumber, gasBailOut *bool) (interface{}, error) {
	block, err := api.blockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	// Prepare base state
	parent, err := api.blockByNumberAndHash(ctx, rpc.BlockNumber(block.NumberU64()-1), block.ParentHash())
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("parent block of block #%d not found", number)
	}

	statedb, release, err := api.backend.StateAtBlock(ctx, parent, defaultTraceReexec, nil, true, false)
	if err != nil {
		return nil, err
	}
	defer release()

	// Native tracers have low overhead
	var (
		txs          = block.Transactions()
		blockHash    = block.Hash()
		blockNumber  = block.NumberU64()
		is158        = api.backend.ChainConfig().IsEIP158(block.Number())
		blockCtx     = core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil)
		signer       = types.MakeSigner(api.backend.ChainConfig(), block.Number(), block.Time())
		posa, isPoSA = api.backend.Engine().(consensus.PoSA)
		txNum        = len(txs)
		rewards      []consensus.Reward
	)

	for _, tx := range txs {
		if isPoSA {
			if isSystemTx, err := posa.IsSystemTransaction(tx, block.Header()); err != nil {
				return nil, err
			} else if isSystemTx {
				txNum--
			}
		}
	}

	out := make([]ParityTrace, 0, len(txs))
	for i, tx := range txs {
		if i == txNum {
			rewards, err = api.backend.Engine().CalculateRewards(api.backend.ChainConfig(), statedb, block.Header(), block.Uncles(), block.Withdrawals(), nil)
			if err != nil {
				return nil, err
			}
		}

		txhash := tx.Hash()
		txpos := uint64(i)
		// Generate the next state snapshot fast without tracing
		msg, _ := core.TransactionToMessage(tx, signer, block.BaseFee())
		txctx := &Context{
			BlockHash:   blockHash,
			BlockNumber: block.Number(),
			TxIndex:     i,
			TxHash:      tx.Hash(),
		}

		res, err := api.traceTx(ctx, msg, txctx, blockCtx, statedb)
		if err != nil {
			return nil, err
		}

		for _, pt := range res.Trace {
			pt.BlockHash = &blockHash
			pt.BlockNumber = &blockNumber
			pt.TransactionHash = &txhash
			pt.TransactionPosition = &txpos
			out = append(out, *pt)
		}

		// Finalize the state so any modifications are written to the trie
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		statedb.Finalise(is158)
	}

	for _, r := range rewards {
		var tr ParityTrace
		rewardAction := &RewardTraceAction{}
		rewardAction.Author = r.Beneficiary
		rewardAction.RewardType = rewardKindToString(r.Kind)
		rewardAction.Value.ToInt().Set(r.Amount)
		tr.Action = rewardAction
		tr.BlockHash = &common.Hash{}
		copy(tr.BlockHash[:], block.Hash().Bytes())
		tr.BlockNumber = new(uint64)
		*tr.BlockNumber = block.NumberU64()
		tr.Type = "reward" // nolint: goconst
		tr.TraceAddress = []int{}
		out = append(out, tr)
	}
	return out, nil
}

func (api *APITrace) Transaction(ctx context.Context, hash common.Hash, gasBailOut *bool) (interface{}, error) {
	tx, blockHash, blockNumber, index, err := api.backend.GetTransaction(ctx, hash)
	if err != nil {
		return nil, err
	}
	// Only mined txes are supported
	if tx == nil {
		return nil, errTxNotFound
	}
	// It shouldn't happen in practice.
	if blockNumber == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	reexec := defaultTraceReexec
	block, err := api.blockByNumberAndHash(ctx, rpc.BlockNumber(blockNumber), blockHash)
	if err != nil {
		return nil, err
	}
	msg, vmctx, statedb, release, err := api.backend.StateAtTransaction(ctx, block, int(index), reexec)
	if err != nil {
		return nil, err
	}
	defer release()

	txctx := &Context{
		BlockHash:   blockHash,
		BlockNumber: block.Number(),
		TxIndex:     int(index),
		TxHash:      hash,
	}

	res, err := api.traceTx(ctx, msg, txctx, vmctx, statedb)
	if err != nil {
		return nil, err
	}
	out := make([]ParityTrace, 0, len(res.Trace))

	for _, pt := range res.Trace {
		pt.BlockHash = &blockHash
		pt.BlockNumber = &blockNumber
		pt.TransactionHash = &hash
		pt.TransactionPosition = &index
		out = append(out, *pt)
	}
	return out, nil
}

func (api *APITrace) Call(ctx context.Context, args TraceCallParam, traceTypes []string, blockNrOrHash rpc.BlockNumberOrHash) (interface{}, error) {
	var (
		err   error
		block *types.Block
	)
	if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = api.blockByHash(ctx, hash)
	} else if number, ok := blockNrOrHash.Number(); ok {
		if number == rpc.PendingBlockNumber {
			return nil, errors.New("tracing on top of pending is not supported")
		}
		block, err = api.blockByNumber(ctx, number)
	} else {
		return nil, errors.New("invalid arguments; neither block nor hash specified")
	}
	if err != nil {
		return nil, err
	}
	// try to recompute the state
	statedb, release, err := api.backend.StateAtBlock(ctx, block, defaultTraceReexec, nil, true, false)
	if err != nil {
		return nil, err
	}
	defer release()

	vmctx := core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil)

	msg, err := args.ToMessage(statedb, api.backend.RPCGasCap(), block.BaseFee())
	if err != nil {
		return nil, err
	}

	var traceTypeTrace, traceTypeStateDiff, traceTypeVmTrace bool
	for _, traceType := range traceTypes {
		switch traceType {
		case TraceTypeTrace:
			traceTypeTrace = true
		case TraceTypeStateDiff:
			traceTypeStateDiff = true
		case TraceTypeVmTrace:
			traceTypeVmTrace = true
		default:
			return nil, fmt.Errorf("unrecognized trace type: %s", traceType)
		}
	}

	res, err := api.traceTx(ctx, &msg, new(Context), vmctx, statedb)
	if err != nil {
		return nil, err
	}

	out := &TraceCallResult{Trace: []*ParityTrace{}}

	out.Output = res.Output

	if traceTypeTrace {
		out.Trace = res.Trace
	} else {
		out.Trace = []*ParityTrace{}
	}
	if traceTypeStateDiff {
		out.StateDiff = res.StateDiff
	}
	if traceTypeVmTrace {
		out.VmTrace = res.VmTrace
	}
	return out, nil
}

func (api *APITrace) CallMany(ctx context.Context, calls json.RawMessage, blockNrOrHash rpc.BlockNumberOrHash) (interface{}, error) {
	var (
		err   error
		block *types.Block
	)
	if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = api.blockByHash(ctx, hash)
	} else if number, ok := blockNrOrHash.Number(); ok {
		if number == rpc.PendingBlockNumber {
			return nil, errors.New("tracing on top of pending is not supported")
		}
		block, err = api.blockByNumber(ctx, number)
	} else {
		return nil, errors.New("invalid arguments; neither block nor hash specified")
	}
	if err != nil {
		return nil, err
	}
	// try to recompute the state
	statedb, release, err := api.backend.StateAtBlock(ctx, block, defaultTraceReexec, nil, true, false)
	if err != nil {
		return nil, err
	}
	defer release()

	vmctx := core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil)

	// parse calls
	var callParams []TraceCallParam
	dec := json.NewDecoder(bytes.NewReader(calls))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if tok != json.Delim('[') {
		return nil, fmt.Errorf("expected array of [callparam, tracetypes]")
	}
	for dec.More() {
		tok, err = dec.Token()
		if err != nil {
			return nil, err
		}
		if tok != json.Delim('[') {
			return nil, fmt.Errorf("expected [callparam, tracetypes]")
		}
		callParams = append(callParams, TraceCallParam{})
		args := &callParams[len(callParams)-1]
		if err = dec.Decode(args); err != nil {
			return nil, err
		}
		if err = dec.Decode(&args.traceTypes); err != nil {
			return nil, err
		}
		tok, err = dec.Token()
		if err != nil {
			return nil, err
		}
		if tok != json.Delim(']') {
			return nil, fmt.Errorf("expected end of [callparam, tracetypes]")
		}
	}

	var traceResult = make([]*TraceCallResult, 0, len(callParams))
	for _, args := range callParams {
		var traceTypeTrace, traceTypeStateDiff, traceTypeVmTrace bool
		for _, traceType := range args.traceTypes {
			switch traceType {
			case TraceTypeTrace:
				traceTypeTrace = true
			case TraceTypeStateDiff:
				traceTypeStateDiff = true
			case TraceTypeVmTrace:
				traceTypeVmTrace = true
			default:
				return nil, fmt.Errorf("unrecognized trace type: %s", traceType)
			}
		}
		msg, err := args.ToMessage(statedb, api.backend.RPCGasCap(), block.BaseFee())
		if err != nil {
			return nil, err
		}

		res, err := api.traceTx(ctx, &msg, new(Context), vmctx, statedb)
		if err != nil {
			return nil, err
		}

		var out = TraceCallResult{}
		out.Output = res.Output

		if traceTypeTrace {
			out.Trace = res.Trace
		} else {
			out.Trace = []*ParityTrace{}
		}
		if traceTypeStateDiff {
			out.StateDiff = res.StateDiff
		}
		if traceTypeVmTrace {
			out.VmTrace = res.VmTrace
		}

		traceResult = append(traceResult, &out)
	}
	return traceResult, nil
}

func (api *APITrace) ReplayTransaction(ctx context.Context, hash common.Hash, traceTypes []string) (interface{}, error) {
	tx, blockHash, blockNumber, index, err := api.backend.GetTransaction(ctx, hash)
	if err != nil {
		return nil, err
	}
	// Only mined txes are supported
	if tx == nil {
		return nil, errTxNotFound
	}
	// It shouldn't happen in practice.
	if blockNumber == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	reexec := defaultTraceReexec
	block, err := api.blockByNumberAndHash(ctx, rpc.BlockNumber(blockNumber), blockHash)
	if err != nil {
		return nil, err
	}
	msg, vmctx, statedb, release, err := api.backend.StateAtTransaction(ctx, block, int(index), reexec)
	if err != nil {
		return nil, err
	}
	defer release()

	var traceTypeTrace, traceTypeStateDiff, traceTypeVmTrace bool
	for _, traceType := range traceTypes {
		switch traceType {
		case TraceTypeTrace:
			traceTypeTrace = true
		case TraceTypeStateDiff:
			traceTypeStateDiff = true
		case TraceTypeVmTrace:
			traceTypeVmTrace = true
		default:
			return nil, fmt.Errorf("unrecognized trace type: %s", traceType)
		}
	}

	txctx := &Context{
		BlockHash:   blockHash,
		BlockNumber: block.Number(),
		TxIndex:     int(index),
		TxHash:      hash,
	}

	res, err := api.traceTx(ctx, msg, txctx, vmctx, statedb)
	if err != nil {
		return nil, err
	}
	out := &TraceCallResult{}
	out.Output = res.Output

	if traceTypeTrace {
		out.Trace = res.Trace
	} else {
		out.Trace = []*ParityTrace{}
	}
	if traceTypeStateDiff {
		out.StateDiff = res.StateDiff
	}
	if traceTypeVmTrace {
		out.VmTrace = res.VmTrace
	}

	return out, nil
}

func (api *APITrace) ReplayBlockTransactions(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash, traceTypes []string) (interface{}, error) {
	var (
		block *types.Block
		err   error
	)
	if hash, ok := blockNrOrHash.Hash(); ok {
		block, err = api.blockByHash(ctx, hash)
	} else if number, ok := blockNrOrHash.Number(); ok {
		if number == rpc.PendingBlockNumber {
			return nil, errors.New("tracing on top of pending is not supported")
		}
		block, err = api.blockByNumber(ctx, number)
	} else {
		return nil, errors.New("invalid arguments; neither block nor hash specified")
	}
	if err != nil {
		return nil, err
	}
	// Prepare base state
	parent, err := api.blockByNumberAndHash(ctx, rpc.BlockNumber(block.NumberU64()-1), block.ParentHash())
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("parent block of block #%d not found", block.NumberU64())
	}

	statedb, release, err := api.backend.StateAtBlock(ctx, parent, defaultTraceReexec, nil, true, false)
	if err != nil {
		return nil, err
	}
	defer release()

	// Native tracers have low overhead
	var (
		txs       = block.Transactions()
		blockHash = block.Hash()
		is158     = api.backend.ChainConfig().IsEIP158(block.Number())
		blockCtx  = core.NewEVMBlockContext(block.Header(), api.chainContext(ctx), nil)
		signer    = types.MakeSigner(api.backend.ChainConfig(), block.Number(), block.Time())

		traceTypeTrace, traceTypeStateDiff, traceTypeVmTrace bool
	)

	for _, traceType := range traceTypes {
		switch traceType {
		case TraceTypeTrace:
			traceTypeTrace = true
		case TraceTypeStateDiff:
			traceTypeStateDiff = true
		case TraceTypeVmTrace:
			traceTypeVmTrace = true
		default:
			return nil, fmt.Errorf("unrecognized trace type: %s", traceType)
		}
	}

	out := make([]*TraceCallResult, len(txs))
	for i, tx := range txs {
		txhash := tx.Hash()
		// Generate the next state snapshot fast without tracing
		msg, _ := core.TransactionToMessage(tx, signer, block.BaseFee())
		txctx := &Context{
			BlockHash:   blockHash,
			BlockNumber: block.Number(),
			TxIndex:     i,
			TxHash:      tx.Hash(),
		}

		res, err := api.traceTx(ctx, msg, txctx, blockCtx, statedb)
		if err != nil {
			return nil, err
		}

		tr := &TraceCallResult{}
		tr.Output = res.Output

		if traceTypeTrace {
			tr.Trace = res.Trace
		} else {
			tr.Trace = []*ParityTrace{}
		}
		if traceTypeStateDiff {
			tr.StateDiff = res.StateDiff
		}
		if traceTypeVmTrace {
			tr.VmTrace = res.VmTrace
		}
		out[i] = tr
		tr.TransactionHash = &txhash

		// Finalize the state so any modifications are written to the trie
		// Only delete empty objects if EIP158/161 (a.k.a Spurious Dragon) is in effect
		statedb.Finalise(is158)
	}
	return out, nil
}

// traceTx configures a new tracer according to the provided configuration, and
// executes the given message in the provided environment. The return value will
// be tracer dependent.
func (api *APITrace) traceTx(ctx context.Context, message *core.Message, txctx *Context, vmctx vm.BlockContext, statedb *state.StateDB) (*TraceCallResult, error) {
	var (
		err       error
		timeout   = defaultTraceTimeout
		txContext = core.NewEVMTxContext(message)
	)

	tracer := newOeTracer(statedb, api.backend.ChainConfig().Rules(vmctx.BlockNumber, vmctx.Random != nil, vmctx.Time))
	vmConfig := vm.Config{
		Tracer:    tracer,
		NoBaseFee: true,
	}
	vmenv := vm.NewEVM(vmctx, txContext, statedb, api.backend.ChainConfig(), vmConfig)
	tracer.setEVM(vmenv)

	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	go func() {
		<-deadlineCtx.Done()
		if errors.Is(deadlineCtx.Err(), context.DeadlineExceeded) {
			tracer.Stop(errors.New("execution timeout"))
			// Stop evm execution. Note cancellation is not necessarily immediate.
			vmenv.Cancel()
		}
	}()
	defer cancel()

	var intrinsicGas uint64 = 0
	// Run the transaction with tracing enabled.
	if posa, ok := api.backend.Engine().(consensus.PoSA); ok && message.From == vmctx.Coinbase &&
		posa.IsSystemContract(message.To) && message.GasPrice.Cmp(big.NewInt(0)) == 0 {
		balance := statedb.GetBalance(consensus.SystemAddress)
		if balance.Cmp(common.Big0) > 0 {
			statedb.SetBalance(consensus.SystemAddress, big.NewInt(0))
			statedb.AddBalance(vmctx.Coinbase, balance)
		}
		intrinsicGas, _ = core.IntrinsicGas(message.Data, message.AccessList, false, true, true, false)
	}

	// Call Prepare to clear out the statedb access list
	statedb.SetTxContext(txctx.TxHash, txctx.TxIndex)
	beforeStatedb := statedb.Copy()
	if _, err = core.ApplyMessage(vmenv, message, new(core.GasPool).AddGas(message.GasLimit)); err != nil {
		return nil, fmt.Errorf("tracing failed: %w", err)
	}

	sdMap := make(map[common.Address]*StateDiffAccount)
	tracer.r.StateDiff = sdMap
	sd := &StateDiff{sdMap: sdMap, evm: vmenv}

	statedb.FinalizeTx(sd)
	sd.CompareStates(beforeStatedb, statedb)

	tracer.CaptureSystemTxEnd(intrinsicGas)
	return tracer.getTraceResult()
}

func (api *APITrace) blockByNumberAndHash(ctx context.Context, number rpc.BlockNumber, hash common.Hash) (*types.Block, error) {
	block, err := api.blockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	if block.Hash() == hash {
		return block, nil
	}
	return api.blockByHash(ctx, hash)
}

// blockByNumber is the wrapper of the chain access function offered by the backend.
// It will return an error if the block is not found.
func (api *APITrace) blockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	block, err := api.backend.BlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", number)
	}
	return block, nil
}

// blockByHash is the wrapper of the chain access function offered by the backend.
// It will return an error if the block is not found.
func (api *APITrace) blockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	block, err := api.backend.BlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block %s not found", hash.Hex())
	}
	return block, nil
}

// chainContext constructs the context reader which is used by the evm for reading
// the necessary chain context.
func (api *APITrace) chainContext(ctx context.Context) core.ChainContext {
	return ethapi.NewChainContext(ctx, api.backend)
}

func rewardKindToString(kind consensus.RewardKind) string {
	switch kind {
	case consensus.RewardAuthor:
		return "block"
	case consensus.RewardEmptyStep:
		return "emptyStep"
	case consensus.RewardExternal:
		return "external"
	case consensus.RewardUncle:
		return "uncle"
	default:
		return "unknown"
	}
}
