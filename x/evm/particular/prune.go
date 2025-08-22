package particular

import (
	"cosmossdk.io/log"
	"fmt"
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
	Hash   common.Hash
}

var Contracts []*Contract

var logger = log.NewLogger(os.Stdout)

func init() {
	erc20ABI, err := abi.JSON(strings.NewReader(WrappedTokenV2ABI))
	if err != nil {
		logger.Error(fmt.Sprintf("parse erc20 contract abi failed:%v", err.Error()))
	}

	storeABI, err := abi.JSON(strings.NewReader(StoreABI))
	if err != nil {
		logger.Error(fmt.Sprintf("parse store abi contract failed:%v", err.Error()))
	}

	ownerABI, err := abi.JSON(strings.NewReader(OwnerABI))
	if err != nil {
		logger.Error(fmt.Sprintf("parse vrf contract abi failed:%v", err.Error()))
	}

	Contracts = []*Contract{
		{
			Addr: common.HexToAddress(WrappedTokenV2Addr),
			Code: WrappedTokenV2Bytecode,
			Abi:  erc20ABI,
		},
		{
			Addr: common.HexToAddress(StoreAddr),
			Code: StoreBytecode,
			Abi:  storeABI,
		},
		{
			Addr: common.HexToAddress(OwnerAddr),
			Code: OwnerBytecode,
			Abi:  ownerABI,
		},
	}
}
