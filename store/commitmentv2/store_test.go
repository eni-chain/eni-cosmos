package commitmentv2

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store/types"
	"github.com/eni-chain/eni-db/sc/memiavl"
	"github.com/stretchr/testify/require"
)

func TestLastCommitID(t *testing.T) {
	tree := memiavl.New(100)
	store := NewStore(tree, log.NewNopLogger())
	require.Equal(t, types.CommitID{Hash: tree.RootHash()}, store.LastCommitID())
}
