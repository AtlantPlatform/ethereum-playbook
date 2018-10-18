package executor

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/AtlantPlatform/ethfw"
	log "github.com/Sirupsen/logrus"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/AtlantPlatform/ethereum-playbook/model"
)

type Executor struct {
	root      *model.Spec
	nodeGroup string

	ethRPC   *rpc.Client
	ethCli   *ethclient.Client
	keycache ethfw.KeyCache
}

func New(ctx model.AppContext, root *model.Spec) (*Executor, error) {
	nodeGroup := ctx.NodeGroup()
	ethRPC, ok := root.Inventory.GetClient(nodeGroup)
	if !ok {
		err := errors.New("no valid RPC client found in the inventory")
		return nil, err
	}
	executor := &Executor{
		root:      root,
		nodeGroup: nodeGroup,
		ethRPC:    ethRPC,
		ethCli:    ethclient.NewClient(ethRPC),
		keycache:  ctx.KeyCache(),
	}
	return executor, nil
}

func (e *Executor) RunCommand(ctx model.AppContext, cmdName string) ([]*CommandResult, bool) {
	if cmdSpec, ok := e.root.CallCmds[cmdName]; ok {
		return e.runCallCmd(ctx, cmdSpec), true
	}
	if cmdSpec, ok := e.root.ReadCmds[cmdName]; ok {
		return e.runReadCmd(ctx, cmdSpec), true
	}
	if cmdSpec, ok := e.root.WriteCmds[cmdName]; ok {
		return e.runWriteCmd(ctx, cmdSpec), true
	}
	return nil, false
}

type CommandResult struct {
	Wallet string
	Result interface{}
	Error  error
}

func replaceWalletPlaceholders(params []interface{}, walletAddress common.Address) []interface{} {
	newParams := append([]interface{}{}, params...)
	for i, param := range newParams {
		if addr, ok := param.(common.Address); ok {
			if bytes.Equal(addr.Bytes(), model.PlaceholderAddr.Bytes()) {
				newParams[i] = walletAddress
			}
		}
	}
	return newParams
}

func replaceReferences(ctx model.AppContext, params []interface{}, root *model.Spec) []interface{} {
	newParams := append([]interface{}{}, params...)
	for i, param := range newParams {
		if ref, ok := param.(*model.WalletFieldReference); ok {
			wallet, _ := root.Wallets.WalletSpec(ref.WalletName)
			walletField := wallet.FieldValue(ref.FieldName)
			switch ref.FieldName {
			case model.WalletSpecBalanceField:
				newParams[i] = walletField.(*big.Int)
			case model.WalletSpecAddressField:
				newParams[i] = common.HexToAddress(walletField.(string))
			default: // as-is
				newParams[i] = walletField
			}
		}
		if arg, ok := param.(*model.ArgReference); ok {
			if arg.ArgID < 0 {
				log.WithField("command", ctx.AppCommand()).Errorln("insufficient arguments provided")
				return nil
			}
			newParams[i] = ctx.AppCommandArgs()[arg.ArgID]
		}
	}
	return newParams
}
