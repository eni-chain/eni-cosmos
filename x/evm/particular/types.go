package particular

import _ "embed"

var (
	//go:embed contract/WrappedTokenV2.bin
	WrappedTokenV2Bytecode string

	//go:embed contract/store.bin
	StoreBytecode string

	//go:embed contract/owner.bin
	OwnerBytecode string
)
