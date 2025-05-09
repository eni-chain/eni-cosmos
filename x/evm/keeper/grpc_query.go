package keeper

import (
	"context"
	"errors"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/cosmos/cosmos-sdk/x/evm/artifacts/native"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
)

var _ types.QueryServer = Querier{}

// Querier defines a wrapper around the x/mint keeper providing gRPC method
// handlers.
type Querier struct {
	*Keeper
}

func NewQuerier(k *Keeper) Querier {
	return Querier{Keeper: k}
}

func (q Querier) EniAddressByEVMAddress(c context.Context, req *types.QueryEniAddressByEVMAddressRequest) (*types.QueryEniAddressByEVMAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	if req.EvmAddress == "" {
		return nil, sdkerrors.ErrInvalidRequest
	}
	evmAddr := common.HexToAddress(req.EvmAddress)
	addr, found := q.Keeper.GetEniAddress(ctx, evmAddr)
	if !found {
		return &types.QueryEniAddressByEVMAddressResponse{Associated: false}, nil
	}

	return &types.QueryEniAddressByEVMAddressResponse{EniAddress: addr.String(), Associated: true}, nil
}

func (q Querier) EVMAddressByEniAddress(c context.Context, req *types.QueryEVMAddressByEniAddressRequest) (*types.QueryEVMAddressByEniAddressResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	if req.EniAddress == "" {
		return nil, sdkerrors.ErrInvalidRequest
	}
	eniAddr, err := sdk.AccAddressFromBech32(req.EniAddress)
	if err != nil {
		return nil, err
	}
	addr, found := q.Keeper.GetEVMAddress(ctx, eniAddr)
	if !found {
		return &types.QueryEVMAddressByEniAddressResponse{Associated: false}, nil
	}

	return &types.QueryEVMAddressByEniAddressResponse{EvmAddress: addr.Hex(), Associated: true}, nil
}

func (q Querier) StaticCall(c context.Context, req *types.QueryStaticCallRequest) (*types.QueryStaticCallResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	if req.To == "" {
		return nil, errors.New("cannot use static call to create contracts")
	}
	if ctx.GasMeter().Limit() == 0 {
		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	}
	to := common.HexToAddress(req.To)
	res, err := q.Keeper.StaticCallEVM(ctx, q.Keeper.AccountKeeper().GetModuleAddress(types.ModuleName), &to, req.Data)
	if err != nil {
		return nil, err
	}
	return &types.QueryStaticCallResponse{Data: res}, nil
}

func (q Querier) Pointer(c context.Context, req *types.QueryPointerRequest) (*types.QueryPointerResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	switch req.PointerType {
	case types.PointerType_NATIVE:
		p, v, e := q.Keeper.GetERC20NativePointer(ctx, req.Pointee)
		return &types.QueryPointerResponse{
			Pointer: p.Hex(),
			Version: uint32(v),
			Exists:  e,
		}, nil

	default:
		return nil, errors.ErrUnsupported
	}
}

func (q Querier) PointerVersion(c context.Context, req *types.QueryPointerVersionRequest) (*types.QueryPointerVersionResponse, error) {
	//ctx := sdk.UnwrapSDKContext(c)
	switch req.PointerType {
	case types.PointerType_NATIVE:
		return &types.QueryPointerVersionResponse{
			Version: uint32(native.CurrentVersion),
		}, nil

	default:
		return nil, errors.ErrUnsupported
	}
}

func (q Querier) Pointee(c context.Context, req *types.QueryPointeeRequest) (*types.QueryPointeeResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	switch req.PointerType {
	case types.PointerType_NATIVE:
		p, v, e := q.Keeper.GetNativePointee(ctx, req.Pointer)
		return &types.QueryPointeeResponse{
			Pointee: p,
			Version: uint32(v),
			Exists:  e,
		}, nil
	default:
		return nil, errors.ErrUnsupported
	}
}
