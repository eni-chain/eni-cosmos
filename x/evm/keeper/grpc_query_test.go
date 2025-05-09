package keeper_test

//
//import (
//	"errors"
//	"testing"
//	"time"
//
//	sdk "github.com/cosmos/cosmos-sdk/types"
//	testkeeper "github.com/cosmos/cosmos-sdk/testutil/keeper"
//
//	"github.com/cosmos/cosmos-sdk/x/evm/artifacts/native"
//	"github.com/cosmos/cosmos-sdk/x/evm/keeper"
//	"github.com/cosmos/cosmos-sdk/x/evm/types"
//	"github.com/stretchr/testify/require"
//)
//
//func TestQueryPointer(t *testing.T) {
//	k := &testkeeper.EVMTestApp.EvmKeeper
//	ctx := testkeeper.EVMTestApp.GetContextForDeliverTx([]byte{}).WithBlockTime(time.Now())
//	eniAddr1, evmAddr1 := testkeeper.MockAddressPair()
//	//eniAddr2, evmAddr2 := testkeeper.MockAddressPair()
//	//eniAddr3, evmAddr3 := testkeeper.MockAddressPair()
//	//eniAddr4, evmAddr4 := testkeeper.MockAddressPair()
//	//eniAddr5, evmAddr5 := testkeeper.MockAddressPair()
//	//eniAddr6, evmAddr6 := testkeeper.MockAddressPair()
//	//eniAddr7, evmAddr7 := testkeeper.MockAddressPair()
//	goCtx := sdk.WrapSDKContext(ctx)
//	k.SetERC20NativePointer(ctx, eniAddr1.String(), evmAddr1)
//
//	q := keeper.Querier{k}
//	res, err := q.Pointer(goCtx, &types.QueryPointerRequest{PointerType: types.PointerType_NATIVE, Pointee: eniAddr1.String()})
//	require.Nil(t, err)
//	require.Equal(t, types.QueryPointerResponse{Pointer: evmAddr1.Hex(), Version: uint32(native.CurrentVersion), Exists: true}, *res)
//	//res, err = q.Pointer(goCtx, &types.QueryPointerRequest{PointerType: types.PointerType_CW20, Pointee: eniAddr2.String()})
//	//require.Nil(t, err)
//	//require.Equal(t, types.QueryPointerResponse{Pointer: evmAddr2.Hex(), Version: uint32(cw20.CurrentVersion(ctx)), Exists: true}, *res)
//	//res, err = q.Pointer(goCtx, &types.QueryPointerRequest{PointerType: types.PointerType_CW721, Pointee: eniAddr3.String()})
//	//require.Nil(t, err)
//	//require.Equal(t, types.QueryPointerResponse{Pointer: evmAddr3.Hex(), Version: uint32(cw721.CurrentVersion), Exists: true}, *res)
//	//res, err = q.Pointer(goCtx, &types.QueryPointerRequest{PointerType: types.PointerType_ERC20, Pointee: evmAddr4.Hex()})
//	//require.Nil(t, err)
//	//require.Equal(t, types.QueryPointerResponse{Pointer: eniAddr4.String(), Version: uint32(erc20.CurrentVersion), Exists: true}, *res)
//	//res, err = q.Pointer(goCtx, &types.QueryPointerRequest{PointerType: types.PointerType_ERC721, Pointee: evmAddr5.Hex()})
//	//require.Nil(t, err)
//	//require.Equal(t, types.QueryPointerResponse{Pointer: eniAddr5.String(), Version: uint32(erc721.CurrentVersion), Exists: true}, *res)
//	//res, err = q.Pointer(goCtx, &types.QueryPointerRequest{PointerType: types.PointerType_CW1155, Pointee: eniAddr6.String()})
//	//require.Nil(t, err)
//	//require.Equal(t, types.QueryPointerResponse{Pointer: evmAddr6.Hex(), Version: uint32(cw1155.CurrentVersion), Exists: true}, *res)
//	//res, err = q.Pointer(goCtx, &types.QueryPointerRequest{PointerType: types.PointerType_ERC1155, Pointee: evmAddr7.Hex()})
//	//require.Nil(t, err)
//	//require.Equal(t, types.QueryPointerResponse{Pointer: eniAddr7.String(), Version: uint32(erc1155.CurrentVersion), Exists: true}, *res)
//}
//
//func TestQueryPointee(t *testing.T) {
//	k, ctx := testkeeper.MockEVMKeeper()
//	_, pointerAddr1 := testkeeper.MockAddressPair()
//	//eniAddr2, evmAddr2 := testkeeper.MockAddressPair()
//	//eniAddr3, evmAddr3 := testkeeper.MockAddressPair()
//	//eniAddr4, evmAddr4 := testkeeper.MockAddressPair()
//	//eniAddr5, evmAddr5 := testkeeper.MockAddressPair()
//	//eniAddr6, evmAddr6 := testkeeper.MockAddressPair()
//	//eniAddr7, evmAddr7 := testkeeper.MockAddressPair()
//	goCtx := sdk.WrapSDKContext(ctx)
//
//	// Set up pointers for each type
//	k.SetERC20NativePointer(ctx, "ufoo", pointerAddr1)
//
//	q := keeper.Querier{k}
//
//	// Test for Native Pointee
//	res, err := q.Pointee(goCtx, &types.QueryPointeeRequest{PointerType: types.PointerType_NATIVE, Pointer: pointerAddr1.Hex()})
//	require.Nil(t, err)
//	require.Equal(t, types.QueryPointeeResponse{Pointee: "ufoo", Version: uint32(native.CurrentVersion), Exists: true}, *res)
//
//	// Test for not registered Native Pointee
//	res, err = q.Pointee(goCtx, &types.QueryPointeeRequest{PointerType: types.PointerType_NATIVE, Pointer: "0x1234567890123456789012345678901234567890"})
//	require.Nil(t, err)
//	require.Equal(t, types.QueryPointeeResponse{Pointee: "", Version: 0, Exists: false}, *res)
//
//	// Test cases for invalid inputs
//	testCases := []struct {
//		name        string
//		req         *types.QueryPointeeRequest
//		expectedRes *types.QueryPointeeResponse
//		expectedErr error
//	}{
//		{
//			name:        "Invalid pointer type",
//			req:         &types.QueryPointeeRequest{PointerType: 999, Pointer: pointerAddr1.Hex()},
//			expectedRes: nil,
//			expectedErr: errors.ErrUnsupported,
//		},
//		{
//			name:        "Empty pointer",
//			req:         &types.QueryPointeeRequest{PointerType: types.PointerType_NATIVE, Pointer: ""},
//			expectedRes: &types.QueryPointeeResponse{Pointee: "", Version: 0, Exists: false},
//			expectedErr: nil,
//		},
//	}
//
//	for _, tc := range testCases {
//		t.Run(tc.name, func(t *testing.T) {
//			res, err := q.Pointee(goCtx, tc.req)
//			if tc.expectedErr != nil {
//				require.ErrorIs(t, err, tc.expectedErr)
//				require.Nil(t, res)
//			} else {
//				require.NoError(t, err)
//				require.Equal(t, tc.expectedRes, res)
//			}
//		})
//	}
//}
