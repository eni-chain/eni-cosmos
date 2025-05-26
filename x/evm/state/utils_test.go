package state_test

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/x/evm/state"
	"github.com/stretchr/testify/require"
)

func TestGetCoinbaseAddress(t *testing.T) {
	coinbaseAddr := state.GetCoinbaseAddress(1).String()
	require.Equal(t, "cosmos1v4mx6hmrda5kucnpwdjsqqqqqqqqqqqp0kuykl", coinbaseAddr)
}
