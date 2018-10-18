package executor

import (
	"errors"
	"fmt"
	"strings"

	"github.com/AtlantPlatform/ethfw"
	log "github.com/Sirupsen/logrus"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/AtlantPlatform/ethereum-playbook/model"
)

func (e *Executor) runWriteCmd(ctx model.AppContext, cmdSpec *model.WriteCmdSpec) []*CommandResult {
	var denominations []string
	for name, contract := range e.root.Contracts {
		for _, instance := range contract.Instances {
			if instance.IsDeployed() {
				binding := instance.BoundContract()
				binding.SetClient(e.ethCli)
				binding.SetAddress(common.HexToAddress(instance.Address))
				contractLog := log.WithFields(log.Fields{
					"contract": name,
					"address":  instance.Address,
				})
				if symbol := instance.FetchTokenSymbol(ctx); len(symbol) > 0 {
					symbol = strings.ToUpper(symbol)
					contractLog.WithField("symbol", symbol).Println("found token symbol")
					denominations = append(denominations, strings.ToLower(symbol))
				}
			}
		}
	}
	var binding *ethfw.BoundContract
	if cmdSpec.Instance != nil {
		binding = cmdSpec.Instance.BoundContract()
		binding.SetClient(e.ethCli)
		// if deployed, the address has been set in loops above
	}
	result := &CommandResult{}
	wallet := cmdSpec.MatchingWallet()
	account := common.HexToAddress(wallet.Address)
	balance, err := e.ethCli.BalanceAt(ctx, account, nil)
	if err != nil {
		result.Error = err
		return []*CommandResult{result}
	}
	wallet.Balance = balance
	gasPrice, _ := e.root.Config.GasPriceInt()
	suggestedGas, err := e.ethCli.SuggestGasPrice(ctx)
	if err == nil && suggestedGas.Cmp(gasPrice) > 0 {
		gasPrice = suggestedGas
	}
	var value model.ExtendedValue
	if len(cmdSpec.Value) > 0 {
		if v, err := cmdSpec.Value.Parse(ctx, e.root, denominations); err != nil {
			result.Error = err
			return []*CommandResult{result}
		} else {
			value.Value = v.Value
			value.Denominator = v.Denominator
		}
	}

	if len(value.Denominator) == 0 && len(cmdSpec.To) > 0 {
		// just send ether
		to := common.HexToAddress(cmdSpec.To)
		callMsg := ethereum.CallMsg{
			From:     account,
			To:       &to,
			Gas:      0,
			GasPrice: gasPrice,
			Value:    value.Value,
			Data:     nil,
		}
		nonce, err := e.ethCli.PendingNonceAt(ctx, account)
		if err != nil {
			result.Error = err
			return []*CommandResult{result}
		}
		gasLimit, _ := e.root.Config.GasLimitInt()
		estimatedGasLimit, err := e.ethCli.EstimateGas(ctx, callMsg)
		if err == nil && estimatedGasLimit < gasLimit {
			gasLimit = estimatedGasLimit
		}
		tx := types.NewTransaction(nonce, to, value.Value, gasLimit, gasPrice, nil)
		pk, ok := e.keycache.PrivateKey(account, wallet.Password)
		if !ok {
			if pk = wallet.PrivKeyECDSA(); pk == nil {
				result.Error = errors.New("failed to get account private key")
				return []*CommandResult{result}
			}
		}
		chainID, _ := e.root.Config.ChainIDInt()
		signer := types.NewEIP155Signer(chainID)
		signedTx, err := types.SignTx(tx, signer, pk)
		if err != nil {
			result.Error = err
			return []*CommandResult{result}
		}
		result.Error = e.ethCli.SendTransaction(ctx, signedTx)
		result.Result = "tx:" + strings.ToLower(signedTx.Hash().Hex())
		return []*CommandResult{result}
	}
	if len(value.Denominator) == 0 && !cmdSpec.Instance.IsDeployed() {
		// need to deploy an instance
		params := replaceWalletPlaceholders(cmdSpec.ParamValues(), account)
		params = replaceReferences(ctx, params, e.root)
		opts := &bind.TransactOpts{
			From:     account,
			Nonce:    nil, // pending state
			Signer:   e.keycache.SignerFn(account, wallet.Password),
			Value:    value.Value,
			GasPrice: gasPrice,
			GasLimit: 0, // estimate
			Context:  ctx,
		}
		contractAddr, tx, err := cmdSpec.Instance.BoundContract().DeployContract(opts, params...)
		if err != nil {
			result.Error = err
			return []*CommandResult{result}
		}
		cmdSpec.Instance.Address = strings.ToLower(contractAddr.Hex())
		cmdSpec.Instance.BoundContract().SetAddress(contractAddr)
		contractLog := log.WithFields(log.Fields{
			"contract": cmdSpec.Instance.Name,
			"address":  cmdSpec.Instance.Address,
		})
		if symbolName := cmdSpec.Instance.FetchTokenSymbol(ctx); len(symbolName) > 0 {
			contractLog.WithField("symbol", strings.ToUpper(symbolName)).Println("fetched token symbol")
		} else {
			contractLog.Println("contract deployed")
		}
		result.Result = "tx:" + strings.ToLower(tx.Hash().Hex())
		return []*CommandResult{result}
	}
	// at this point, contract is deployed and we just want to use its method
	var params []interface{}
	if len(value.Denominator) > 0 {
		instance, ok := e.root.Contracts.FindByTokenSymbol(value.Denominator)
		if !ok {
			result.Error = fmt.Errorf("referenced token contract not found: %s", value.Denominator)
			return []*CommandResult{result}
		} else if !instance.IsDeployed() {
			result.Error = fmt.Errorf("referenced token contract is not deployed yet: %s", value.Denominator)
			return []*CommandResult{result}
		}
		// override binding with other referenced contract
		binding = instance.BoundContract()
		if len(cmdSpec.To) == 0 {
			result.Error = errors.New("no transfer recipient address specified")
			return []*CommandResult{result}
		}
		to := common.HexToAddress(cmdSpec.To)
		cmdSpec.Method = "transfer"
		params = []interface{}{to, value.Value}
	} else {
		params = replaceWalletPlaceholders(cmdSpec.ParamValues(), account)
		params = replaceReferences(ctx, params, e.root)
	}
	opts := &bind.TransactOpts{
		From:     account,
		Nonce:    nil, // pending state
		Signer:   e.keycache.SignerFn(account, wallet.Password),
		GasPrice: gasPrice,
		GasLimit: 0, // estimate
		Context:  ctx,
	}
	tx, err := binding.Transact(opts, cmdSpec.Method, params...)
	if err != nil {
		result.Error = err
		return []*CommandResult{result}
	}
	result.Result = "tx:" + strings.ToLower(tx.Hash().Hex())
	return []*CommandResult{result}
}
