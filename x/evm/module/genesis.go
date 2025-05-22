package evm

import (
	"math/big"
	"strings"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/cosmos/cosmos-sdk/x/evm/keeper"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
)

var (
// evmModuleAddress = "eni1vqu8rska6swzdmnhf90zuv0xmelej4lqdj955g"
)

// InitGenesis initializes the module's state from a provided genesis state.
func InitGenesis(ctx sdk.Context, k *keeper.Keeper, genState types.GenesisState) {
	k.InitGenesis(ctx, genState)
	k.SetParams(ctx, genState.Params)
	initEniAddressList := strings.Split(genState.Params.InitEniAddress, ",")
	initEniAmountList := strings.Split(genState.Params.InitEniAmount, ",")
	if len(initEniAddressList) != len(initEniAmountList) {
		panic("InitEniAddress and InitEniAmount length not equal")
	}
	//The balance allocated to the evm module will be reallocated to the address we designed
	evmSupply := uint256.MustFromDecimal("0")
	for i, initEniAddress := range initEniAddressList {
		initEniAmount := initEniAmountList[i]
		if len(initEniAddress) == 0 || len(initEniAmount) == 0 {
			continue
		}
		amt := uint256.MustFromDecimal(initEniAmount)
		SetBalance(ctx, k, common.HexToAddress(initEniAddress), amt)
		evmSupply.Add(evmSupply, amt)
	}

	evmModuleAddress := k.AccountKeeper().GetModuleAccount(ctx, types.ModuleName).GetAddress()
	balance := k.GetBalance(ctx, evmModuleAddress)
	if balance.Cmp(evmSupply.ToBig()) != 0 {
		panic("The balance of mint must be equal to that of mint in the bank module ")
	} else { //Reset the balance allocated to the evm module to zero
		err := k.BankKeeper().SetBalance(ctx, evmModuleAddress, sdk.NewCoin(k.GetBaseDenom(ctx), math.NewInt(0)))
		if err != nil {
			panic(err)
		}
	}
	balance = k.GetBalance(ctx, evmModuleAddress)
}
func send(ctx sdk.Context, k *keeper.Keeper, from sdk.AccAddress, to sdk.AccAddress, amt *big.Int) error {
	ueni := sdk.NewCoin(k.GetBaseDenom(ctx), math.NewIntFromBigIntMut(amt))
	err := k.BankKeeper().SendCoins(ctx, from, to, sdk.NewCoins(ueni))
	if err != nil {
		return err
	}
	k.Logger().Info("genesis send", "from", from, "to", to, "amount", ueni)
	return nil
}
func getBalance(ctx sdk.Context, k *keeper.Keeper, evmAddr common.Address) *big.Int {
	eniAddr := k.GetEniAddressOrDefault(ctx, evmAddr)
	return k.GetBalance(ctx, eniAddr)
}
func SetBalance(ctx sdk.Context, k *keeper.Keeper, evmAddr common.Address, amt *uint256.Int) {

	eniAddr := k.GetEniAddressOrDefault(ctx, evmAddr)
	//moduleAddr := k.AccountKeeper().GetModuleAddress(types.ModuleName)
	//err := send(ctx, k, eniAddr, moduleAddr, getBalance(ctx, k, evmAddr))
	//if err != nil {
	//	panic(err)
	//}
	a := amt.ToBig()
	//coinsAmt := sdk.NewCoins(sdk.NewCoin(k.GetBaseDenom(ctx), math.NewIntFromBigIntMut(a)))
	err := k.BankKeeper().SetBalance(ctx, eniAddr, sdk.NewCoin(k.GetBaseDenom(ctx), math.NewIntFromBigIntMut(a)))
	//if err = k.BankKeeper().MintCoins(ctx, types.ModuleName, coinsAmt); err != nil {
	//	panic(err)
	//}
	//err = send(ctx, k, moduleAddr, eniAddr, a)
	if err != nil {
		panic(err)
	}
}

// ExportGenesis returns the module's exported genesis.
func ExportGenesis(ctx sdk.Context, k *keeper.Keeper) *types.GenesisState {
	genesis := types.DefaultGenesis()
	genesis.Params = k.GetParams(ctx)

	// this line is used by starport scaffolding # genesis/module/export

	return genesis
}
