package executor

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/AtlantPlatform/ethereum-playbook/model"
)

func (e *Executor) runCallCmd(ctx model.AppContext, cmdSpec *model.CallCmdSpec) []*CommandResult {
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
			result.Error = e.ethRPC.CallContext(ctx, &result.Result, cmdSpec.Method, params...)
			results[offset] = result
		}
		return results
	}
	result := &CommandResult{}
	params := replaceReferences(ctx, cmdSpec.ParamValues(), e.root)
	result.Error = e.ethRPC.CallContext(ctx, &result.Result, cmdSpec.Method, params...)
	results = append(results, result)
	return results
}
