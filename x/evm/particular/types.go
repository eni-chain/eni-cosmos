package particular

import _ "embed"

var (
	//go:embed contract/erc20.bin
	Erc20Bytecode string

	//go:embed contract/store.bin
	StoreBytecode string

	//go:embed contract/owner.bin
	OwnerBytecode string
)
