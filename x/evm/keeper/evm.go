package keeper

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	cosmossdk_io_math "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/cosmos/cosmos-sdk/utils"
	"github.com/cosmos/cosmos-sdk/utils/metrics"
	"github.com/cosmos/cosmos-sdk/x/evm/state"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
)

type EVMCallFunc func(caller vm.ContractRef, addr *common.Address, input []byte, gas uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error)

var MaxUint64BigInt = new(big.Int).SetUint64(math.MaxUint64)

func (k *Keeper) HandleInternalEVMCall(ctx sdk.Context, req *types.MsgInternalEVMCall) (*sdk.Result, error) {
	var to *common.Address
	if req.To != "" {
		addr := common.HexToAddress(req.To)
		to = &addr
	}
	senderAddr, err := sdk.AccAddressFromBech32(req.Sender)
	if err != nil {
		return nil, err
	}
	ret, err := k.CallEVM(ctx, k.GetEVMAddressOrDefault(ctx, senderAddr), to, req.Value, req.Data)
	if err != nil {
		return nil, err
	}
	return &sdk.Result{Data: ret}, nil
}

func (k *Keeper) HandleInternalEVMDelegateCall(ctx sdk.Context, req *types.MsgInternalEVMDelegateCall) (*sdk.Result, error) {
	var to *common.Address
	if req.To != "" {
		addr := common.HexToAddress(req.To)
		to = &addr
	} else {
		return nil, errors.New("cannot use a CosmWasm contract to delegate-create an EVM contract")
	}
	addr, _, exists := k.GetPointerInfo(ctx, types.PointerReverseRegistryKey(common.BytesToAddress([]byte(req.FromContract))))
	if !exists || common.BytesToAddress(addr).Cmp(*to) != 0 {
		return nil, errors.New("only pointer contract can make delegatecalls")
	}
	zeroInt := cosmossdk_io_math.ZeroInt()
	senderAddr, err := sdk.AccAddressFromBech32(req.Sender)
	if err != nil {
		return nil, err
	}
	// delegatecall caller must be associated; otherwise any state change on EVM contract will be lost
	// after they asssociate.
	senderEvmAddr, found := k.GetEVMAddress(ctx, senderAddr)
	if !found {
		err := types.NewAssociationMissingErr(req.Sender)
		metrics.IncrementAssociationError("evm_handle_internal_evm_delegate_call", err)
		return nil, err
	}
	ret, err := k.CallEVM(ctx, senderEvmAddr, to, &zeroInt, req.Data)
	if err != nil {
		return nil, err
	}
	return &sdk.Result{Data: ret}, nil
}

func (k *Keeper) CallEVM(ctx sdk.Context, from common.Address, to *common.Address, val *cosmossdk_io_math.Int, data []byte) (retdata []byte, reterr error) {
	// TODO: dev
	//if ctx.IsEVM() && !ctx.EVMEntryViaWasmdPrecompile() {
	//	return nil, errors.New("eni does not support EVM->CW->EVM call pattern")
	//}
	if to == nil && len(data) > params.MaxInitCodeSize {
		return nil, fmt.Errorf("%w: code size %v, limit %v", core.ErrMaxInitCodeSizeExceeded, len(data), params.MaxInitCodeSize)
	}
	value := utils.Big0
	if val != nil {
		if val.IsNegative() {
			return nil, sdkerrors.ErrInvalidCoins
		}
		value = val.BigInt()
	}
	// This call was not part of an existing StateTransition, so it should trigger one
	executionCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()) //.WithEVMEntryViaWasmdPrecompile(false)
	stateDB := state.NewDBImpl(executionCtx, k, false)
	gp := k.GetGasPool()
	evmMsg := &core.Message{
		Nonce:            stateDB.GetNonce(from), // replay attack is prevented by the AccountSequence number set on the CW transaction that triggered this call
		GasLimit:         k.getEvmGasLimitFromCtx(ctx),
		GasPrice:         utils.Big0, // fees are already paid on the CW transaction
		GasFeeCap:        utils.Big0,
		GasTipCap:        utils.Big0,
		To:               to,
		Value:            value,
		Data:             data,
		SkipFromEOACheck: false,
		SkipNonceChecks:  false,
		From:             from,
	}
	res, err := k.applyEVMMessage(ctx, evmMsg, stateDB, gp)
	if err != nil {
		return nil, err
	}
	k.consumeEvmGas(ctx, res.UsedGas)
	if res.Err != nil {
		return nil, res.Err
	}
	surplus, err := stateDB.Finalize()
	if err != nil {
		return nil, err
	}
	vmErr := ""
	if res.Err != nil {
		vmErr = res.Err.Error()
	}
	existingReceipt, err := k.GetTransientReceipt(ctx, ctx.TxSum())
	if err == nil {
		for _, l := range existingReceipt.Logs {
			stateDB.AddLog(&ethtypes.Log{
				Address: common.HexToAddress(l.Address),
				Topics:  utils.Map(l.Topics, common.HexToHash),
				Data:    l.Data,
			})
		}
		if existingReceipt.VmError != "" {
			vmErr = fmt.Sprintf("%s\n%s\n", existingReceipt.VmError, vmErr)
		}
	}
	existingDeferredInfo, found := k.GetEVMTxDeferredInfo(ctx)
	if found {
		surplus = surplus.Add(existingDeferredInfo.Surplus)
	}
	receipt, err := k.WriteReceipt(ctx, stateDB, evmMsg, ethtypes.LegacyTxType, ctx.TxSum(), res.UsedGas, vmErr)
	if err != nil {
		return nil, err
	}
	bloom := ethtypes.Bloom{}
	bloom.SetBytes(receipt.LogsBloom)
	k.AppendToEvmTxDeferredInfo(ctx, bloom, ctx.TxSum(), surplus)
	return res.ReturnData, nil
}

func (k *Keeper) StaticCallEVM(ctx sdk.Context, from sdk.AccAddress, to *common.Address, data []byte) ([]byte, error) {
	evm, err := k.createReadOnlyEVM(ctx, from)
	if err != nil {
		return nil, err
	}
	return k.callEVM(ctx, k.GetEVMAddressOrDefault(ctx, from), to, nil, data, func(caller vm.ContractRef, addr *common.Address, input []byte, gas uint64, _ *big.Int) ([]byte, uint64, error) {
		return evm.StaticCall(caller, *addr, input, gas)
	})
}

func (k *Keeper) callEVM(ctx sdk.Context, from common.Address, to *common.Address, val *cosmossdk_io_math.Int, data []byte, f EVMCallFunc) ([]byte, error) {
	evmGasLimit := k.getEvmGasLimitFromCtx(ctx)
	value := utils.Big0
	if val != nil {
		value = val.BigInt()
	}
	ret, leftoverGas, err := f(vm.AccountRef(from), to, data, evmGasLimit, value)
	k.consumeEvmGas(ctx, evmGasLimit-leftoverGas)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

// only used for StaticCalls
func (k *Keeper) createReadOnlyEVM(ctx sdk.Context, from sdk.AccAddress) (*vm.EVM, error) {
	executionCtx := ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	if ctx.GasMeter() != nil {
		executionCtx = ctx.WithGasMeter(ctx.GasMeter())
	}

	stateDB := state.NewDBImpl(executionCtx, k, true)
	gp := k.GetGasPool()
	blockCtx, err := k.GetVMBlockContext(executionCtx, gp)
	if err != nil {
		return nil, err
	}
	cfg := types.DefaultChainConfig().EthereumConfig(k.ChainID(ctx))
	txCtx := vm.TxContext{Origin: k.GetEVMAddressOrDefault(ctx, from)}
	evm := vm.NewEVM(*blockCtx, stateDB, cfg, vm.Config{})
	evm.SetTxContext(txCtx)
	return evm, nil
}

func (k *Keeper) getEvmGasLimitFromCtx(ctx sdk.Context) uint64 {
	eniGasRemaining := ctx.GasMeter().Limit() - ctx.GasMeter().GasConsumedToLimit()
	if ctx.GasMeter().Limit() <= 0 {
		return math.MaxUint64
	}
	evmGasBig := cosmossdk_io_math.LegacyNewDecFromInt(cosmossdk_io_math.NewIntFromUint64(eniGasRemaining)).Quo(k.GetPriorityNormalizer(ctx.WithGasMeter(storetypes.NewInfiniteGasMeter()))).TruncateInt().BigInt()
	if evmGasBig.Cmp(MaxUint64BigInt) > 0 {
		evmGasBig = MaxUint64BigInt
	}
	return evmGasBig.Uint64()
}

func (k *Keeper) consumeEvmGas(ctx sdk.Context, usedEvmGas uint64) {
	ctx.GasMeter().ConsumeGas(cosmossdk_io_math.LegacyNewDecFromInt(cosmossdk_io_math.NewIntFromUint64(usedEvmGas)).Mul(k.GetPriorityNormalizer(ctx)).TruncateInt().Uint64(), "call EVM")
}
