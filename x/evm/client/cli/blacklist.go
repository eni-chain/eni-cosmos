package cli

import (
	"errors"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/x/evm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/spf13/cobra"
)

// NewDelBlacklistsCmd returns a CLI command handler for creating a MsgSend transaction.
func NewDelBlacklistsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del-blacklists ",
		Short: "del blacklists for evm module",
		Long: `Update the blacklists configuration of the evm module.
`,
		Args: cobra.MaximumNArgs(10),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			for _, arg := range args {
				if !common.IsHexAddress(arg) {
					return errors.New("invalid address")
				}
			}

			msg := &types.MsgDelBlackLists{
				Authority: clientCtx.GetFromAddress().String(),
				Addresses: args,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

// NewAddBlacklistsCmd returns a CLI command handler for creating a MsgSend transaction.
func NewAddBlacklistsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-blacklists",
		Short: "add blacklists for evm module",
		Long: `Update the blacklists configuration of the evm module.
`,
		Args: cobra.MaximumNArgs(10),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			for _, arg := range args {
				if !common.IsHexAddress(arg) {
					return errors.New("invalid address")
				}
			}
			msg := &types.MsgAddBlackLists{
				Authority: clientCtx.GetFromAddress().String(),
				Addresses: args,
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
