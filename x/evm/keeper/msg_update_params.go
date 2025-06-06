package keeper

import (
	"context"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
)

var (
	gasParamsManager = "eni16hwsrl7v23mnf08ymu844n622sruyawn23dmd3"
)

func (msg msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	if gasParamsManager != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidGasManager, "invalid gas manager; expected %s, got %s", gasParamsManager, req.Authority)
	}

	if req.Params.BaseFeePerGas.GT(req.Params.MaximumFeePerGas) {
		return nil, errorsmod.Wrapf(types.ErrInvalidGasManager, "MaximumFeePerGas must greater than BaseFeePerGas ")
	}

	WBaseFee := math.LegacyNewDec(10000)
	if req.Params.MaxDynamicBaseFeeUpwardAdjustment.GT(WBaseFee) {
		return nil, errorsmod.Wrapf(types.ErrInvalidGasManager, "MaxDynamicBaseFeeUpwardAdjustment must less than 10k ")
	}
	if req.Params.MaxDynamicBaseFeeDownwardAdjustment.GT(WBaseFee) {
		return nil, errorsmod.Wrapf(types.ErrInvalidGasManager, "MaxDynamicBaseFeeDownwardAdjustment must less than 10k  ")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	params := msg.GetParams(ctx)
	ZeroBaseFee := math.LegacyNewDec(0)
	if req.Params.BaseFeePerGas.GT(ZeroBaseFee) {
		params.MinimumFeePerGas = req.Params.MinimumFeePerGas
		params.BaseFeePerGas = req.Params.BaseFeePerGas
		params.MaximumFeePerGas = req.Params.MaximumFeePerGas
	}

	if req.Params.MaxDynamicBaseFeeUpwardAdjustment.GT(ZeroBaseFee) {
		params.MaxDynamicBaseFeeUpwardAdjustment = req.Params.MaxDynamicBaseFeeUpwardAdjustment
	}

	if req.Params.MaxDynamicBaseFeeDownwardAdjustment.GT(ZeroBaseFee) {
		params.MaxDynamicBaseFeeDownwardAdjustment = req.Params.MaxDynamicBaseFeeDownwardAdjustment
	}

	if req.Params.TargetGasUsedPerBlock > 0 {
		params.TargetGasUsedPerBlock = req.Params.TargetGasUsedPerBlock
	}

	msg.SetParams(ctx, params)

	return &types.MsgUpdateParamsResponse{}, nil
}
