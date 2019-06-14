// Copyright 2017-2019 Tensigma Ltd. All rights reserved.
// Use of this source code is governed by Microsoft Reference Source
// License (MS-RSL) that can be found in the LICENSE file.

// Package sol provides a convenient interface for calling the 'solc' Solidity Compiler from Go.
package sol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type Contract struct {
	Name            string
	SourcePath      string
	CompilerVersion string
	Address         common.Address

	ABI []byte
	Bin string
}

type Compiler interface {
	SetAllowPaths(paths []string) Compiler
	Compile(prefix, path string, optimize int) (map[string]*Contract, error)
}

func NewSolCompiler(solcPath string) (Compiler, error) {
	s := &solCompiler{
		solcPath: solcPath,
	}
	if err := s.verify(); err != nil {
		return nil, err
	}
	return s, nil
}

type solCompiler struct {
	solcPath   string
	allowPaths []string
}

func (s *solCompiler) verify() error {
	out, err := exec.Command(s.solcPath, "--version").CombinedOutput()
	if err != nil {
		err = fmt.Errorf("solc verify: failed to exec solc: %v", err)
		return err
	}
	hasPrefix := strings.HasPrefix(string(out), "solc, the solidity compiler")
	if !hasPrefix {
		err := fmt.Errorf("solc verify: executable output was unexpected (output: %s)", out)
		return err
	}
	return nil
}

func (s *solCompiler) SetAllowPaths(paths []string) Compiler {
	s.allowPaths = paths
	return s
}

type solcContract struct {
	ABI string `json:"abi"`
	Bin string `json:"bin"`
}

type solcOutput struct {
	Contracts map[string]solcContract `json:"contracts"`
	Version   string                  `json:"version"`
}

func (s *solCompiler) Compile(prefix, path string, optimize int) (map[string]*Contract, error) {
	args := []string{s.solcPath}
	if len(s.allowPaths) > 0 {
		args = append(args, "--allow-paths", strings.Join(s.allowPaths, ","))
	}
	args = append(args, "--combined-json", "bin,abi", filepath.Join(prefix, path))
	if optimize > 0 {
		args = append(args, "--optimize", fmt.Sprintf("--optimize-runs=%d", optimize))
	}
	cmd := exec.Cmd{
		Path:   s.solcPath,
		Args:   args,
		Dir:    prefix,
		Stderr: os.Stderr,
	}
	out, err := cmd.Output()
	if err != nil {
		err = fmt.Errorf("solc: failed to compile contract: %v", err)
		return nil, err
	}
	var result solcOutput
	if err := json.Unmarshal(out, &result); err != nil {
		err = fmt.Errorf("solc: failed to unmarshal JSON output: %v", err)
		return nil, err
	}
	if len(result.Contracts) == 0 {
		err := errors.New("solc: no contracts compiled")
		return nil, err
	}
	contracts := make(map[string]*Contract, len(result.Contracts))
	for id, c := range result.Contracts {
		idParts := strings.Split(id, ":")
		if len(idParts) == 1 {
			err := fmt.Errorf("solc: found an unnamed contract in output: %s", id)
			return nil, err
		}
		name := idParts[len(idParts)-1]
		if err != nil {
			err := fmt.Errorf("solc: failed to remarshal ABI: %v", err)
			return nil, err
		}
		contracts[name] = &Contract{
			Name:            name,
			SourcePath:      idParts[0],
			CompilerVersion: result.Version,

			ABI: []byte(c.ABI),
			Bin: c.Bin,
		}
	}
	return contracts, nil
}

func WhichSolc() (string, error) {
	out, err := exec.Command("which", "solc").Output()
	if err != nil {
		return "", errors.New("solc executable file not found in $PATH")
	}
	return string(bytes.TrimSpace(out)), nil
}
