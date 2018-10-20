package executor

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/ethereum/go-ethereum/common"

	"github.com/AtlantPlatform/ethereum-playbook/model"
)

func (e *Executor) runTarget(ctx model.AppContext,
	targetName string, target model.TargetSpec) chan []*CommandResult {

	out := make(chan []*CommandResult, 100)
	defer close(out)

	for _, targetCmd := range target {
		cmdName := targetCmd.Name()
		if cmdSpec, ok := e.root.CallCmds[cmdName]; ok {
			results := e.runCallCmd(ctx, cmdSpec)
			out <- setName(results, cmdName)
		} else if cmdSpec, ok := e.root.ReadCmds[cmdName]; ok {
			results := e.runReadCmd(ctx, cmdSpec)
			out <- setName(results, cmdName)
		} else if cmdSpec, ok := e.root.WriteCmds[cmdName]; ok {
			execLog := log.WithFields(log.Fields{
				"target":  targetName,
				"command": cmdName,
			})
			results := e.runWriteCmd(ctx, cmdSpec)
			out <- setName(results, cmdName)
			if len(results) == 0 || results[0].Error != nil {
				execLog.Errorln("stopping target execution — tx sumbit failed")
				return out
			}
			if !targetCmd.IsDeferred() {
				awaitTimeout, _ := e.root.Config.AwaitTimeoutDuration()
				execLog.WithFields(log.Fields{
					// "handle":  results[0].Result,
					"timeout": awaitTimeout.String(),
				}).Infoln("awaiting write command transaction")
				awaitCtx, cancelFn := context.WithTimeout(ctx, awaitTimeout)
				if err := e.awaitTx(awaitCtx, results[0].Result); err != nil {
					execLog.WithError(err).Errorln("stopping target execution after await")
					cancelFn()
					return out
				}
				cancelFn()
			}
		}
	}
	return out
}

func setName(results []*CommandResult, name string) []*CommandResult {
	if len(results) == 0 {
		return []*CommandResult{{
			Name: name,
		}}
	}
	for i := range results {
		results[i].Name = name
	}
	return results
}

func (e *Executor) awaitTx(ctx context.Context, v interface{}) error {
	value, ok := v.(string)
	if !ok {
		err := fmt.Errorf("unknown result type: %T", v)
		return err
	}
	if strings.HasPrefix(value, "tx:") {
		value = value[3:]
	} else if !strings.HasPrefix(value, "0x") {
		err := fmt.Errorf("value is not a hex-string: %s", value)
		return err
	}

	tx, isPending, err := e.ethCli.TransactionByHash(ctx, common.HexToHash(value))
	if err != nil {
		return err
	} else if !isPending {
		return nil
	}
	t := time.NewTimer(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			_, isPending, err = e.ethCli.TransactionByHash(ctx, tx.Hash())
			if err == nil && !isPending {
				receipt, err := e.ethCli.TransactionReceipt(ctx, tx.Hash())
				if err != nil {
					return err
				} else if status := receipt.Status; status == 0 {
					err := errors.New("transction execution ended with failing status code")
					return err
				}
				// finally a transaction receipt,
				// with a successful status
				return nil
			} else if err != nil {
				log.WithError(err).Warningln("error while checking the transaction status")
				t.Reset(10 * time.Second)
				continue
			}
			t.Reset(time.Second)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
