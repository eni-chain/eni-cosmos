package config

import (
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/spf13/cast"
)

type Config struct {
	// system contract admin address,address must is evm address
	ContractAdminAddr string `mapstructure:"contract_admin_addr"`
	// Skipping the starting height of signature check caused by validator node shutdown for slashing module
	SlashingSkipHeight int64 `mapstructure:"slashing_skip_height"`
	// GasParams update manager address,address must is eni address for evm module
	GasParamsManager string `mapstructure:"gas_params_manager"`
	//Black Lists Enable Height
	BlackListsEnableHeight int64 `mapstructure:"black_lists_enable_height"`
	FixEvmReceiveHeight    int64 `mapstructure:"fix_evm_receive_height"`
}

var DefaultUpdateConfig = &Config{
	ContractAdminAddr:      "0x110b6FB6675Fb2a310394ac3a43b23Fc23aB9BC6",
	SlashingSkipHeight:     0,
	GasParamsManager:       "eni1wklu5t7ctecdlfr465lm6ms709xneg0rf45ajt",
	BlackListsEnableHeight: 0,
	FixEvmReceiveHeight:    0,
}

const (
	flagContractAdminAddr      = "update.contract_admin_addr"
	flagSlashingSkipHeight     = "update.slashing_skip_height"
	flagGasAdminAddr           = "update.gas_admin_addr"
	flagBlackListsEnableHeight = "update.black_lists_enable_height"
	flagFixEvmReceiveHeight    = "update.fix_evm_receive_height"
)

func ReadConfig(opts servertypes.AppOptions) (*Config, error) {
	cfg := DefaultUpdateConfig // copy
	var err error
	if v := opts.Get(flagContractAdminAddr); v != nil {
		if cfg.ContractAdminAddr = cast.ToString(v); err != nil {
			return cfg, err
		}
	}
	if v := opts.Get(flagSlashingSkipHeight); v != nil {
		if cfg.SlashingSkipHeight, err = cast.ToInt64E(v); err != nil {
			return cfg, err
		}
	}
	if v := opts.Get(flagGasAdminAddr); v != nil {
		if cfg.GasParamsManager = cast.ToString(v); err != nil {
			return cfg, err
		}
	}
	if v := opts.Get(flagBlackListsEnableHeight); v != nil {
		if cfg.BlackListsEnableHeight, err = cast.ToInt64E(v); err != nil {
			return cfg, err
		}
	}
	if v := opts.Get(flagFixEvmReceiveHeight); v != nil {
		if cfg.FixEvmReceiveHeight, err = cast.ToInt64E(v); err != nil {
			return cfg, err
		}
	}

	return cfg, nil
}
