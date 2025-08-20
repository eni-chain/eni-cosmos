package particular

import (
	"cosmossdk.io/log"
	"encoding/hex"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/evm/keeper"
	particular "github.com/cosmos/cosmos-sdk/x/evm/particular/sdk"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"os"
	"strings"
)

type Contract struct {
	Addr   common.Address
	Code   string
	Abi    abi.ABI
	Pruned []byte
}

var Contracts []*Contract

var logger = log.NewLogger(os.Stdout)

func init() {
	erc20ABI, err := abi.JSON(strings.NewReader(particular.ERC20ABI))
	if err != nil {
		logger.Error(fmt.Sprintf("parse erc20 contract abi failed:%v", err.Error()))
	}

	storeABI, err := abi.JSON(strings.NewReader(particular.StoreABI))
	if err != nil {
		logger.Error(fmt.Sprintf("parse store abi contract failed:%v", err.Error()))
	}

	ownerABI, err := abi.JSON(strings.NewReader(particular.OwnerABI))
	if err != nil {
		logger.Error(fmt.Sprintf("parse vrf contract abi failed:%v", err.Error()))
	}

	Contracts = []*Contract{
		{
			Addr: common.HexToAddress(particular.Erc20Addr),
			Code: Erc20Bytecode,
			Abi:  erc20ABI,
		},
		{
			Addr: common.HexToAddress(particular.StoreAddr),
			Code: StoreBytecode,
			Abi:  storeABI,
		},
		{
			Addr: common.HexToAddress(particular.OwnerAddr),
			Code: OwnerBytecode,
			Abi:  ownerABI,
		},
	}
}

func Prune(ctx sdk.Context, evmKeeper *keeper.Keeper) {
	if Contracts == nil {
		evmKeeper.Logger().Info("empty contracts config", "height", ctx.BlockHeight())
		return
	}

	evmKeeper.Logger().Info(fmt.Sprintf("pruning particular contracts at height %d", ctx.BlockHeight()))

	caller := evmKeeper.AccountKeeper().GetModuleAddress(authtypes.FeeCollectorName)
	for _, contract := range Contracts {
		evmKeeper.Logger().Info(fmt.Sprintf("pruning contract %s", contract.Addr.String()))

		code, err := hex.DecodeString(strings.TrimSpace(contract.Code))
		if err != nil {
			panic(fmt.Errorf("failed to decode new contract code: %s", err.Error()))
		}

		body, err := evmKeeper.CallEVM(ctx, common.Address(caller), nil, nil, code)
		if err != nil {
			panic(fmt.Errorf("failed to execute contract constructor: %s", err.Error()))
		}
		contract.Pruned = body

		//todo: wait real particular contract for next operation
		//evmKeeper.SetCode(ctx, contract.Addr, body)
		//calldata, err := contract.Abi.Pack("init", abi.Argument{})
		//if err != nil {
		//	panic(fmt.Errorf("failed to pack calldata: %s", err.Error()))
		//}

		//_, err = evmKeeper.CallEVM(ctx, common.Address(caller), &contract.Addr, nil, calldata)
		//if err != nil {
		//	panic(fmt.Errorf("failed to execute contract init: %s", err.Error()))
		//}
	}
}
