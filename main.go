package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

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
	specPath  = flag.String("f", "ethereum-playbook.yml", "Custom path to ethereum-playbook.yml spec file.")
	solcPath  = flag.String("s", "solc", "Name or path of Solidity compiler (solc, not solcjs).")
	nodeGroup = flag.String("g", "genesis", "Inventory group name, corresponding to Geth nodes.")
	printHelp = flag.Bool("h", false, "Print help.")
)

func init() {
	app.StringOpt("f", "ethereum-playbook.yml", "Custom path to ethereum-playbook.yml spec file.")
	app.StringOpt("s", "solc", "Name or path of Solidity compiler (solc, not solcjs).")
	app.StringOpt("g", "genesis", "Inventory group name, corresponding to Geth nodes.")
	app.BoolOpt("h", false, "Print help.")
}

func main() {
	flag.Parse()
	spec, ok := loadSpec()
	if !ok {
		if *printHelp {
			flag.Usage()
			os.Exit(0)
		}
		os.Exit(-1)
	}
	registerCommands(app, spec)
	app.Before = func() {
		if *printHelp {
			app.PrintLongHelp()
			os.Exit(0)
		}
	}
	app.Action = func() {
		validateSpec(spec, "", nil)
		log.Infoln("spec validated")
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func registerCommands(app *cli.Cli, spec *model.Spec) {
	callCmdNames := make([]string, 0, len(spec.CallCmds))
	for name := range spec.CallCmds {
		callCmdNames = append(callCmdNames, name)
	}
	sort.Sort(sort.StringSlice(callCmdNames))
	for _, name := range callCmdNames {
		cmd, _ := spec.CallCmds.CallCmdSpec(name)
		desc := cmd.Description
		args := cmd.ArgCount()
		if len(desc) == 0 {
			desc = fmt.Sprintf("Generic CALL command, accepts %d args", args)
		}
		app.Command(name, desc, newCommand(spec, name, cmd.ArgCount()))
	}

	readCmdNames := make([]string, 0, len(spec.ReadCmds))
	for name := range spec.ReadCmds {
		readCmdNames = append(readCmdNames, name)
	}
	sort.Sort(sort.StringSlice(readCmdNames))
	for _, name := range readCmdNames {
		cmd, _ := spec.ReadCmds.ReadCmdSpec(name)
		desc := cmd.Description
		args := cmd.ArgCount()
		if len(desc) == 0 {
			desc = fmt.Sprintf("Generic READ command, accepts %d args", args)
		}
		app.Command(name, desc, newCommand(spec, name, cmd.ArgCount()))
	}

	writeCmdNames := make([]string, 0, len(spec.WriteCmds))
	for name := range spec.WriteCmds {
		writeCmdNames = append(writeCmdNames, name)
	}
	sort.Sort(sort.StringSlice(writeCmdNames))
	for _, name := range writeCmdNames {
		cmd, _ := spec.WriteCmds.WriteCmdSpec(name)
		desc := cmd.Description
		args := cmd.ArgCount()
		if len(desc) == 0 {
			desc = fmt.Sprintf("Generic WRITE command, accepts %d args", args)
		}
		app.Command(name, desc, newCommand(spec, name, cmd.ArgCount()))
	}
}

func newCommand(spec *model.Spec, name string, argCount int) cli.CmdInitializer {
	return func(cmd *cli.Cmd) {
		args := make([]*string, argCount)
		for i := 0; i < argCount; i++ {
			args[i] = cmd.StringArg(fmt.Sprintf("ARG%d", i+1), "", fmt.Sprintf("Command argument $%d", i+1))
		}
		cmd.Action = func() {
			appArgs := []string{name}
			for _, arg := range args {
				appArgs = append(appArgs, *arg)
			}
			ctx := validateSpec(spec, name, appArgs)
			cmdLog := log.WithFields(log.Fields{
				"command": name,
			})
			executor, err := executor.New(ctx, spec)
			if err != nil {
				cmdLog.WithError(err).Fatalln("failed to init executor")
			}
			results, found := executor.RunCommand(ctx, name)
			if !found {
				cmdLog.Fatalln("command not found")
			}
			exportResults(spec, results)
		}
	}
}

func loadSpec() (*model.Spec, bool) {
	var spec *model.Spec
	specLog := log.WithFields(log.Fields{
		"filename": *specPath,
	})
	specData, err := ioutil.ReadFile(*specPath)
	if err != nil {
		specLog.WithError(err).Errorln("failed to load spec file")
		return nil, false
	}
	if err := yaml.Unmarshal(specData, &spec); err != nil {
		specLog.WithError(err).Errorln("failed to parse YAML in the spec file")
		return nil, false
	}
	absSpecPath, err := filepath.Abs(*specPath)
	if err != nil {
		specLog.WithError(err).Errorln("failed to get absolute path of the spec file")
		return nil, false
	}
	if spec.Config == nil {
		spec.Config = model.DefaultConfigSpec
	}
	spec.Config.SpecDir = filepath.Dir(absSpecPath)
	return spec, true
}

func validateSpec(spec *model.Spec, appCommand string, appArgs []string) model.AppContext {
	specLog := log.WithFields(log.Fields{
		"filename": *specPath,
	})
	var solcCompiler sol.Compiler
	if spec.Contracts.UseSolc() {
		solcAbsPath, err := exec.LookPath(*solcPath)
		if err != nil {
			solcAbsPath = *solcPath
		}
		compiler, err := sol.NewSolCompiler(solcAbsPath)
		if err != nil {
			specLog.WithError(err).Fatalln("spec uses .sol contracts, but no solc compiler found")
		}
		solcCompiler = compiler
	}
	ctx := model.NewAppContext(context.Background(), appCommand, appArgs, *nodeGroup,
		spec.Config.SpecDir, solcCompiler, ethfw.NewKeyCache())
	if ok := spec.Validate(ctx); !ok {
		os.Exit(-1)
	}
	return ctx
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
		v, err := json.MarshalIndent(prettify(result.Result), "", "\t")
		if err != nil {
			panic(err)
		}
		fmt.Printf("%s (@%s): %s\n", result.Wallet, walletName, v)
	}
}

func jsonPrint(v interface{}) {
	vv, err := json.MarshalIndent(v, "", "\t")
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
