package types

import (
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

// GetEvmSender returns the sender address of the transaction.
func (m *MsgEVMTransaction) GetEvmSender() (common.Address, error) {
	txData, err := UnpackTxData(m.Data)
	if err != nil {
		return common.Address{}, err
	}
	ethTx := ethtypes.NewTx(txData.AsEthereumData())
	chainID := ethTx.ChainId()
	signer := ethtypes.LatestSignerForChainID(chainID)
	sender, err := signer.Sender(ethTx)
	if err != nil {
		return common.Address{}, err
	}
	return sender, nil
}
