package state

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
)

func (s *DBImpl) GetNonce(addr common.Address) uint64 {

	return s.k.GetNonce(s.ctx, addr)
}

func (s *DBImpl) SetNonce(addr common.Address, nonce uint64, reason tracing.NonceChangeReason) {

	if s.logger != nil && s.logger.OnNonceChange != nil {
		// The SetCode method could be modified to return the old code/hash directly.
		s.logger.OnNonceChange(addr, s.GetNonce(addr), nonce)
	}

	s.k.SetNonce(s.ctx, addr, nonce)
}
