package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/AtlantPlatform/ethfw"
	"github.com/AtlantPlatform/ethfw/sol"
	log "github.com/Sirupsen/logrus"
	cli "github.com/jawher/mow.cli"

	"github.com/AtlantPlatform/ethereum-playbook/executor"
	"github.com/AtlantPlatform/ethereum-playbook/model"
	"github.com/AtlantPlatform/ethereum-playbook/yaml"
)

var app = cli.App("ethereum-playbook", "Ethereum contracts deployment and management tool.")

var (
	specPath   = app.StringOpt("f file", "ethereum-playbook.yml", "Custom path to ethereum-playbook.yml spec file.")
	solcPath   = app.StringOpt("s solc", "solc", "Name or path of Solidity compiler (solc, not solcjs).")
	nodeGroup  = app.StringOpt("g group", "genesis", "Inventory group name, corresponding to Geth nodes.")
	appCommand = app.StringArg("COMMAND", "", "Specify a command or target to run. If empty, will only verify spec.")
)

func main() {
	app.Spec = "[-f] [-s] [-g] [COMMAND]"
	app.Action = func() {
		var spec *model.Spec
		mainLog := log.WithFields(log.Fields{
			"filename": *specPath,
		})
		specData, err := ioutil.ReadFile(*specPath)
		if err != nil {
			mainLog.WithError(err).Fatalln("failed to load spec file")
		}
		if err := yaml.Unmarshal(specData, &spec); err != nil {
			mainLog.WithError(err).Fatalln("failed to parse YAML in the spec file")
		}
		var solcCompiler sol.Compiler
		if spec.Contracts.UseSolc() {
			solcAbsPath, err := exec.LookPath(*solcPath)
			if err != nil {
				solcAbsPath = *solcPath
			}
			compiler, err := sol.NewSolCompiler(solcAbsPath)
			if err != nil {
				mainLog.WithError(err).Fatalln("spec uses .sol contracts, but no solc compiler found")
			}
			solcCompiler = compiler
		}
		absSpecPath, err := filepath.Abs(*specPath)
		if err != nil {
			mainLog.WithError(err).Fatalln("failed to get absolute path of the spec file")
		}
		specDir := filepath.Dir(absSpecPath)
		ctx := model.NewAppContext(context.Background(), *appCommand, *nodeGroup,
			specDir, solcCompiler, ethfw.NewKeyCache())
		if ok := spec.Validate(ctx); !ok {
			os.Exit(-1)
		}
		log.Infoln("config validated")
		executor, err := executor.New(ctx, spec)
		if err != nil {
			mainLog.WithError(err).Fatalln("failed to init executor")
		}
		if len(*appCommand) > 0 {
			results, found := executor.RunCommand(ctx, *appCommand)
			if !found {
				log.WithField("command", *appCommand).Fatalln("command not found")
			}
			exportResults(spec, results)
		}
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func exportResults(spec *model.Spec, results []*executor.CommandResult) {
	if len(results) == 0 {
		jsonPrint(&ErrorObject{Error: "no results"})
		return
	} else if len(results) == 1 {
		if len(results[0].Wallet) == 0 {
			if results[0].Error != nil {
				jsonPrint(&ErrorObject{Error: results[0].Error.Error()})
				return
			}
			jsonPrint(prettify(results[0].Result))
			return
		}
	}
	for _, result := range results {
		walletName := spec.Wallets.NameOf(result.Wallet)
		if result.Error != nil {
			v, err := json.Marshal(&ErrorObject{Error: result.Error.Error()})
			if err != nil {
				panic(err)
			}
			fmt.Printf("%s (@%s): %s\n", result.Wallet, walletName, v)
			continue
		}
		v, err := json.Marshal(prettify(result.Result))
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s (@%s): %s\n", result.Wallet, walletName, v)
	}
}

func jsonPrint(v interface{}) {
	vv, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(vv))
}

type ErrorObject struct {
	Error string `json: "error"`
}

func yamlPrint(v interface{}) {
	vv, err := yaml.Marshal(v)
	if err != nil {
		panic(err)
	}
	log.Println(string(vv))
}
