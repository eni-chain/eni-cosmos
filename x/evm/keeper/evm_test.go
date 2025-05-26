package keeper_test

import (
	"math/big"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evm/artifacts/erc20"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	testkeeper "github.com/cosmos/cosmos-sdk/testutil/keeper"
	"github.com/cosmos/cosmos-sdk/x/evm/artifacts/native"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

func TestInternalCallCreateContract(t *testing.T) {
	bytecode := native.GetBin()
	abi, err := native.NativeMetaData.GetAbi()
	require.Nil(t, err)
	args, err := abi.Pack("", "test", "TST", "TST", uint8(6))
	require.Nil(t, err)
	contractData := append(bytecode, args...)

	ctx, k := createTestContext(t)
	ctx = ctx.WithBlockHeight(2).WithTxSum([32]byte{1, 2, 3})

	testAddr, _ := testkeeper.MockAddressPair()
	amt := sdk.NewCoins(sdk.NewCoin(k.GetBaseDenom(ctx), math.NewInt(200000000)))
	require.Nil(t, k.BankKeeper().MintCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(k.GetBaseDenom(ctx), math.NewInt(200000000)))))
	require.Nil(t, k.BankKeeper().SendCoinsFromModuleToAccount(ctx, types.ModuleName, testAddr, amt))
	req := &types.MsgInternalEVMCall{
		Sender: testAddr.String(),
		Data:   contractData,
	}
	// circular interop call
	ctx = ctx.WithIsEVM(true).WithMultiStore(ctx.MultiStore().CacheMultiStore())
	_, err = k.HandleInternalEVMCall(ctx, req)
	require.Nil(t, err)
	ctx = ctx.WithIsEVM(false)
	oldBaseFee := k.GetCurrBaseFeePerGas(ctx)
	k.SetCurrBaseFeePerGas(ctx, math.LegacyZeroDec())
	_, err = k.HandleInternalEVMCall(ctx, req)
	require.Nil(t, err)
	receipt, err := k.GetReceipt(ctx, [32]byte{1, 2, 3})
	require.Nil(t, err)
	require.NotNil(t, receipt)
	// reset base fee
	k.SetCurrBaseFeePerGas(ctx, oldBaseFee)
}

func TestInternalCall(t *testing.T) {
	bytecode := erc20.GetBin()
	abi, err := erc20.Erc20MetaData.GetAbi()
	require.Nil(t, err)
	args, err := abi.Pack("", "test", "TST")
	require.Nil(t, err)
	contractData := append(bytecode, args...)

	// 1. create
	ctx, k := createTestContext(t)
	ctx = ctx.WithBlockHeight(2)
	testAddr, senderEvmAddr := testkeeper.MockAddressPair()
	k.SetAddressMapping(ctx, testAddr, senderEvmAddr)
	req := &types.MsgInternalEVMCall{
		Sender: testAddr.String(),
		Data:   contractData,
	}
	ctx = ctx.WithIsEVM(true).WithMultiStore(ctx.MultiStore().CacheMultiStore())
	ret, err := k.HandleInternalEVMCall(ctx, req)
	require.Nil(t, err)

	// 2. mint
	contractAddr := crypto.CreateAddress(senderEvmAddr, 0)
	require.NotEmpty(t, k.GetCode(ctx, contractAddr))
	require.Equal(t, ret.Data, k.GetCode(ctx, contractAddr))
	args, err = abi.Pack("mint", senderEvmAddr, big.NewInt(10000))
	require.Nil(t, err)
	req = &types.MsgInternalEVMCall{
		Sender: testAddr.String(),
		To:     contractAddr.Hex(),
		Data:   args,
	}
	_, err = k.HandleInternalEVMCall(ctx, req)
	require.Nil(t, err)

	// 3. transfer
	receiverAddr, evmAddr := testkeeper.MockAddressPair()
	k.SetAddressMapping(ctx, receiverAddr, evmAddr)
	args, err = abi.Pack("transfer", evmAddr, big.NewInt(1000))
	require.Nil(t, err)
	req = &types.MsgInternalEVMCall{
		Sender: testAddr.String(),
		To:     contractAddr.Hex(),
		Data:   args,
	}
	_, err = k.HandleInternalEVMCall(ctx, req)
	require.Nil(t, err)

	// 4.query
	args, err = abi.Pack("balanceOf", evmAddr)
	require.Nil(t, err)
	req = &types.MsgInternalEVMCall{
		Sender: testAddr.String(),
		To:     contractAddr.Hex(),
		Data:   args,
	}
	ret, err = k.HandleInternalEVMCall(ctx, req)
	require.Nil(t, err)
	balance := new(big.Int).SetBytes(ret.Data)
	require.Equal(t, "1000", balance.String())
}

func TestStaticCall(t *testing.T) {
	bytecode := erc20.GetBin()
	abi, err := erc20.Erc20MetaData.GetAbi()
	require.Nil(t, err)
	args, err := abi.Pack("", "test", "TST")
	require.Nil(t, err)
	contractData := append(bytecode, args...)

	// 1. create
	ctx, k := createTestContext(t)
	ctx = ctx.WithBlockHeight(2)
	testAddr, senderEvmAddr := testkeeper.MockAddressPair()
	k.SetAddressMapping(ctx, testAddr, senderEvmAddr)
	req := &types.MsgInternalEVMCall{
		Sender: testAddr.String(),
		Data:   contractData,
	}
	ctx = ctx.WithIsEVM(true).WithMultiStore(ctx.MultiStore().CacheMultiStore())
	ret, err := k.HandleInternalEVMCall(ctx, req)
	require.Nil(t, err)

	// 2. mint
	contractAddr := crypto.CreateAddress(senderEvmAddr, 0)
	require.NotEmpty(t, k.GetCode(ctx, contractAddr))
	require.Equal(t, ret.Data, k.GetCode(ctx, contractAddr))
	args, err = abi.Pack("mint", senderEvmAddr, big.NewInt(10000))
	require.Nil(t, err)
	req = &types.MsgInternalEVMCall{
		Sender: testAddr.String(),
		To:     contractAddr.Hex(),
		Data:   args,
	}
	_, err = k.HandleInternalEVMCall(ctx, req)
	require.Nil(t, err)

	// 3. transfer
	receiverAddr, evmAddr := testkeeper.MockAddressPair()
	k.SetAddressMapping(ctx, receiverAddr, evmAddr)
	args, err = abi.Pack("transfer", evmAddr, big.NewInt(1000))
	require.Nil(t, err)
	req = &types.MsgInternalEVMCall{
		Sender: testAddr.String(),
		To:     contractAddr.Hex(),
		Data:   args,
	}
	_, err = k.HandleInternalEVMCall(ctx, req)
	require.Nil(t, err)

	// 4.query
	args, err = abi.Pack("balanceOf", evmAddr)
	require.Nil(t, err)

	res, err := k.StaticCallEVM(ctx, testAddr, &contractAddr, args)
	require.Nil(t, err)
	decoded, err := abi.Unpack("balanceOf", res)
	require.Nil(t, err)
	require.Equal(t, 1, len(decoded))
	require.Equal(t, big.NewInt(int64(1000)), decoded[0].(*big.Int))
}
