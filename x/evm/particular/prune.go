package particular

import (
	"cosmossdk.io/log"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"os"
	"strings"
)

type InitArgs struct {
	Name   string
	Symbol string
	Supply string
	Holder common.Address
}

type Contract struct {
	Addr common.Address
	Code string
	Abi  abi.ABI
	Body []byte
	Hash common.Hash
	Args InitArgs
}

var Contracts []*Contract

var logger = log.NewLogger(os.Stdout)

func init() {
	erc20ABI, err := abi.JSON(strings.NewReader(WrappedTokenV2ABI))
	if err != nil {
		logger.Error(fmt.Sprintf("parse erc20 contract abi failed:%v", err.Error()))
	}

	//storeABI, err := abi.JSON(strings.NewReader(StoreABI))
	//if err != nil {
	//	logger.Error(fmt.Sprintf("parse store abi contract failed:%v", err.Error()))
	//}

	//ownerABI, err := abi.JSON(strings.NewReader(OwnerABI))
	//if err != nil {
	//	logger.Error(fmt.Sprintf("parse vrf contract abi failed:%v", err.Error()))
	//}

	Contracts = []*Contract{
		{
			Addr: common.HexToAddress(EniPegUSDTAddr),
			Code: WrappedTokenV2Bytecode,
			Abi:  erc20ABI,
			Args: InitArgs{
				Name:   EniPegUSDTName,
				Symbol: EniPegUSDTSymbol,
				Supply: EniPegUSDTSupply,
				Holder: common.HexToAddress(EniPegUSDTHolder),
			},
		},
		{
			Addr: common.HexToAddress(EniPegUSDCAddr),
			Code: WrappedTokenV2Bytecode,
			Abi:  erc20ABI,
			Args: InitArgs{
				Name:   EniPegUSDCName,
				Symbol: EniPegUSDCSymbol,
				Supply: EniPegUSDCSupply,
				Holder: common.HexToAddress(EniPegUSDCHolder),
			},
		},
		{
			Addr: common.HexToAddress(EniPegBTCAddr),
			Code: WrappedTokenV2Bytecode,
			Abi:  erc20ABI,
			Args: InitArgs{
				Name:   EniPegBTCName,
				Symbol: EniPegBTCSymbol,
				Supply: EniPegBTCSupply,
				Holder: common.HexToAddress(EniPegBTCHolder),
			},
		},
		{
			Addr: common.HexToAddress(EniPegETHAddr),
			Code: WrappedTokenV2Bytecode,
			Abi:  erc20ABI,
			Args: InitArgs{
				Name:   EniPegETHName,
				Symbol: EniPegETHSymbol,
				Supply: EniPegETHSupply,
				Holder: common.HexToAddress(EniPegETHHolder),
			},
		},
		{
			Addr: common.HexToAddress(EniPegBNBAddr),
			Code: WrappedTokenV2Bytecode,
			Abi:  erc20ABI,
			Args: InitArgs{
				Name:   EniPegBNBName,
				Symbol: EniPegBNBSymbol,
				Supply: EniPegBNBSupply,
				Holder: common.HexToAddress(EniPegBNBHolder),
			},
		},
		{
			Addr: common.HexToAddress(EniPegSOLAddr),
			Code: WrappedTokenV2Bytecode,
			Abi:  erc20ABI,
			Args: InitArgs{
				Name:   EniPegSOLName,
				Symbol: EniPegSOLSymbol,
				Supply: EniPegSOLSupply,
				Holder: common.HexToAddress(EniPegSOLHolder),
			},
		},
		//{
		//	Addr: common.HexToAddress(StoreAddr),
		//	Code: StoreBytecode,
		//	Abi:  storeABI,
		//},
		//{
		//	Addr: common.HexToAddress(OwnerAddr),
		//	Code: OwnerBytecode,
		//	Abi:  ownerABI,
		//},
	}
}
