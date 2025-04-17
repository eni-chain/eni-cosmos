package rootmultiv2

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store/types"
	"github.com/eni-chain/eni-db/config"
	"github.com/stretchr/testify/require"
)

func TestLastCommitID(t *testing.T) {
	store := NewStore(t.TempDir(), log.NewNopLogger(), config.StateCommitConfig{}, config.StateStoreConfig{}, false)
	require.Equal(t, types.CommitID{}, store.LastCommitID())
}
