package executor

import (
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/AtlantPlatform/ethereum-playbook/model"
)

func (e *Executor) runReadCmd(ctx model.AppContext, cmdSpec *model.ReadCmdSpec) []*CommandResult {
	if !cmdSpec.Instance.IsDeployed() {
		return []*CommandResult{{
			Error: errors.New("contract instance is not deployed yet"),
		}}
	}
	binding := cmdSpec.Instance.BoundContract()
	binding.SetClient(e.ethCli)
	binding.SetAddress(common.HexToAddress(cmdSpec.Instance.Address))
	matchingWallets := cmdSpec.MatchingWallets()
	results := make([]*CommandResult, len(matchingWallets))
	if len(matchingWallets) > 0 {
		for offset, walletSpec := range matchingWallets {
			walletAddress := common.HexToAddress(walletSpec.Address)
			params := replaceWalletPlaceholders(cmdSpec.ParamValues(), walletAddress)
			params = replaceReferences(ctx, params, e.root)
			result := &CommandResult{
				Wallet: walletSpec.Address,
			}
			opts := &bind.CallOpts{
				From:    walletAddress,
				Context: ctx,
			}
			result.Error = binding.Call(opts, &result.Result, cmdSpec.Method, params...)
			results[offset] = result
		}
		return results
	}
	opts := &bind.CallOpts{
		Context: ctx,
	}
	result := &CommandResult{}
	params := replaceReferences(ctx, cmdSpec.ParamValues(), e.root)
	result.Error = binding.Call(opts, &result.Result, cmdSpec.Method, params...)
	results = append(results, result)
	return results
}
