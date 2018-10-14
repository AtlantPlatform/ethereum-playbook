package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/AtlantPlatform/ethfw"
	"github.com/AtlantPlatform/ethfw/sol"
	log "github.com/Sirupsen/logrus"
	cli "github.com/jawher/mow.cli"

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
		jsonPrint(spec)
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func jsonPrint(v interface{}) {
	vv, _ := json.MarshalIndent(v, "", "\t")
	log.Println(string(vv))
}
