package executor

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"

	"github.com/AtlantPlatform/ethereum-playbook/model"
)

func (e *Executor) runViewCmd(ctx model.AppContext, cmdSpec *model.ViewCmdSpec) []*CommandResult {
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
			if err := binding.Call(opts, &result.Result, cmdSpec.Method, params...); err != nil {
				if strings.HasPrefix(err.Error(), "abi: cannot unmarshal tuple") {
					storage := newValStorage()
					result.Error = binding.Call(opts, &storage.pointers, cmdSpec.Method, params...)
					result.Result = storage.Trim()
				} else {
					result.Error = err
				}
			}
			results[offset] = result
		}
		return results
	}
	result := &CommandResult{}
	params := replaceReferences(ctx, cmdSpec.ParamValues(), e.root)
	opts := &bind.CallOpts{
		Context: ctx,
	}
	if err := binding.Call(opts, &result.Result, cmdSpec.Method, params...); err != nil {
		if strings.HasPrefix(err.Error(), "abi: cannot unmarshal tuple") {
			storage := newValStorage()
			result.Error = binding.Call(opts, &storage.pointers, cmdSpec.Method, params...)
			result.Result = storage.Trim()
		} else {
			result.Error = err
		}
	}
	results = append(results, result)
	return results
}

type valStorage struct {
	backing  [maxReturnValues]interface{}
	pointers [maxReturnValues]*interface{}
}

const maxReturnValues = 32

func newValStorage() *valStorage {
	storage := &valStorage{}
	for i := 0; i < maxReturnValues; i++ {
		storage.pointers[i] = &storage.backing[i]
	}
	return storage
}

func (storage *valStorage) Trim() []interface{} {
	last := 0
	for i := 0; i < maxReturnValues; i++ {
		if storage.backing[i] != nil {
			last = i
		}
	}
	return storage.backing[:last+1]
}
