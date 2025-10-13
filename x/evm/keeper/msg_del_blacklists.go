package keeper

import (
	"context"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/utils/config"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
)

func (msg msgServer) DelBlackLists(goCtx context.Context, req *types.MsgDelBlackLists) (*types.MsgDelBlackListsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	gasParamsManager := config.DefaultUpdateConfig.GasParamsManager

	if gasParamsManager != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidGasManager, "invalid gas manager; expected %s, got %s", gasParamsManager, req.Authority)
	}

	// test get request
	msg.logger.Info("DelBlackLists function called ", "address ")
	kv := msg.Keeper.PrefixStore(ctx, types.BlackListsPrefix)
	for _, addr := range req.Addresses {
		if common.IsHexAddress(addr) {
			evmAddr := common.HexToAddress(addr)
			oldData := kv.Get(evmAddr[:])
			if oldData == nil {
				return nil, errorsmod.Wrapf(types.ErrBlackListsUpdate, "the address %s is not on the blacklists ", evmAddr)
			}
			kv.Delete(evmAddr[:])
		} else {
			return nil, errorsmod.Wrapf(types.ErrBlackListsUpdate, "req Addresses not evmAddress %s", addr)
		}
	}
	return &types.MsgDelBlackListsResponse{}, nil
}
