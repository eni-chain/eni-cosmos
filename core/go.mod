module cosmossdk.io/core

go 1.20

require (
	cosmossdk.io/api v0.7.0
	cosmossdk.io/depinject v1.0.0
	cosmossdk.io/math v1.1.2
	github.com/stretchr/testify v1.9.0
	google.golang.org/grpc v1.64.1
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/cosmos/cosmos-proto v1.0.0-beta.5 // indirect
	github.com/cosmos/gogoproto v1.5.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/tendermint/go-amino v0.16.0 // indirect
	golang.org/x/exp v0.0.0-20231006140011-7918f672742d // indirect
	golang.org/x/net v0.27.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240709173604-40e1e62336c5 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	sigs.k8s.io/yaml v1.4.0 // indirect
)

// Version tagged too early and incompatible with v0.50 (latest at the time of tagging)
retract v0.12.0
