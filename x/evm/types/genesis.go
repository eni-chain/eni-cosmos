package types

// this line is used by starport scaffolding # genesis/types/import

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		// this line is used by starport scaffolding # genesis/types/default
		Params:              DefaultParams(),
		AddressAssociations: make([]*AddressAssociation, 0),
		Codes:               make([]*Code, 0),
		States:              make([]*ContractState, 0),
		Nonces:              make([]*Nonce, 0),
		Serialized:          make([]*Serialized, 0),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// this line is used by starport scaffolding # genesis/types/validate

	return gs.Params.Validate()
}
