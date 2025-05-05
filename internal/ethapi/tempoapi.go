package ethapi

import (
	"fmt"
	"math"
	"context"
	"time"

    "github.com/ethereum/go-ethereum/rpc"
    "github.com/ethereum/go-ethereum/common/hexutil"
    "github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/gasestimator"
	"github.com/ethereum/go-ethereum/internal/ethapi/override"
)

func NewTempoAPI(apiBackend Backend) *TempoAPI {
    return &TempoAPI{
        b: apiBackend,
    }
}

type TempoAPI struct {
    b Backend
}

// Call executes the given transaction on the state for the given block number, but the signature check of tempo is skipped.
 //
 // Additionally, the caller can specify a batch of contract for fields overriding.
 //
 // Note, this function doesn't make and changes in the state/blockchain and is
 // useful to execute and retrieve values.
 func (s *TempoAPI) Eth_call(ctx context.Context, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides) (hexutil.Bytes, error) {
	result, err := s.Tempo_DoCall(ctx, s.b, args, blockNrOrHash, overrides, blockOverrides, s.b.RPCEVMTimeout(), s.b.RPCGasCap())
	// log an error message that says test
	if err != nil {
		return nil, err
	}
	// If the result contains a revert reason, try to unpack and return it.
	if len(result.Revert()) > 0 {
		return nil, newRevertError(result.Revert())
	}
	return result.Return(), result.Err
}

func (s *TempoAPI) Tempo_DoCall(ctx context.Context, b Backend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *override.StateOverride, blockOverrides *override.BlockOverrides, timeout time.Duration, globalGasCap uint64) (*core.ExecutionResult, error) {
	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	state, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return nil, err
	}
	if err := overrides.Apply(state, nil); err != nil {
		return nil, err
	}

	// this block modifies the deployed bytecode of a contract before the call and sets it back after the call
	codeBefore := state.GetCode(tempo_contractAddress)
	// modified deployed bytecode of Tempo - get it in /bin/contracts/RookSwap.json -> deployedBytecode
	state.SetCode(tempo_contractAddress, tempo_manipulatedBytecode)
	// make sure to set the code back to the original
	defer state.SetCode(tempo_contractAddress, codeBefore)

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	// Get a new instance of the EVM.
	msg:= args.ToMessage(header.BaseFee, false, false)
	blockCtx := core.NewEVMBlockContext(header, NewChainContext(ctx, b), nil)
	if blockOverrides != nil {
		blockOverrides.Apply(&blockCtx)
	}
	evm := vm.NewEVM(blockCtx, state, b.ChainConfig(), vm.Config{NoBaseFee: true})

	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	// Execute the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	result, err := core.ApplyMessage(evm, msg, gp)
	if err := state.Error(); err != nil {
		return nil, err
	}

	// If the timer caused an abort, return an appropriate error message
	if evm.Cancelled() {
		return nil, fmt.Errorf("execution aborted (timeout = %v)", timeout)
	}
	if err != nil {
		return result, fmt.Errorf("err: %w (supplied gas %d)", err, msg.GasLimit)
	}

	return result, nil
}

// EstimateGas returns an estimate of the amount of gas needed to execute the
// given transaction against the current pending block.
func (s *TempoAPI) Eth_estimateGas(ctx context.Context, args TransactionArgs, blockNrOrHash *rpc.BlockNumberOrHash, overrides *override.StateOverride) (hexutil.Uint64, error) {
	bNrOrHash := rpc.BlockNumberOrHashWithNumber(rpc.LatestBlockNumber)
	if blockNrOrHash != nil {
		bNrOrHash = *blockNrOrHash
	}
	return s.Tempo_DoEstimateGas(ctx, s.b, args, bNrOrHash, overrides, s.b.RPCGasCap())
}

// Tempo_DoEstimateGas returns the lowest possible gas limit that allows the transaction to run
// successfully at block `blockNrOrHash`. It returns error if the transaction would revert, or if
// there are unexpected failures. The gas limit is capped by both `args.Gas` (if non-nil &
// non-zero) and `gasCap` (if non-zero).
func (s *TempoAPI) Tempo_DoEstimateGas(ctx context.Context, b Backend, args TransactionArgs, blockNrOrHash rpc.BlockNumberOrHash, overrides *override.StateOverride, gasCap uint64) (hexutil.Uint64, error) {
	// Retrieve the base state and mutate it with any overrides
	state, header, err := b.StateAndHeaderByNumberOrHash(ctx, blockNrOrHash)
	if state == nil || err != nil {
		return 0, err
	}
	if err = overrides.Apply(state, nil); err != nil {
		return 0, err
	}

	// this block modifies the deployed bytecode of a contract before the call and sets it back after the call
	codeBefore := state.GetCode(tempo_contractAddress)
	// modified deployed bytecode of Tempo - get it in /bin/contracts/RookSwap.json -> deployedBytecode
	state.SetCode(tempo_contractAddress, tempo_manipulatedBytecode)
	// make sure to set the code back to the original
	defer state.SetCode(tempo_contractAddress, codeBefore)

	// Construct the gas estimator option from the user input
	opts := &gasestimator.Options{
		Config:     b.ChainConfig(),
		Chain:      NewChainContext(ctx, b),
		Header:     header,
		State:      state,
		ErrorRatio: estimateGasErrorRatio,
	}
	// Run the gas estimation andwrap any revertals into a custom return
	call := args.ToMessage(header.BaseFee, false, false)
	estimate, revert, err := gasestimator.Estimate(ctx, call, opts, gasCap)
	if err != nil {
		if len(revert) > 0 {
			return 0, newRevertError(revert)
		}
		return 0, err
	}
	return hexutil.Uint64(estimate), nil
}
