// Copyright 2017, 2018 Tensigma Ltd. All rights reserved.
// Use of this source code is governed by Microsoft Reference Source
// License (MS-RSL) that can be found in the LICENSE file.

package gasmeter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/AtlantPlatform/ethfw"
)

type GasStation interface {
	Estimate(priority GasPriority) (*ethfw.Wei, time.Duration)
}

type GasPriority string

const (
	GasPrioritySafeLow GasPriority = "safe"
	GasPriorityFast    GasPriority = "fast"
	GasPriorityFastest GasPriority = "fastest"
)

type gasStation struct {
	baseURL string

	statsMux *sync.RWMutex
	blockNum uint64

	safeLowGas *ethfw.Wei
	fastGas    *ethfw.Wei
	fastestGas *ethfw.Wei

	safeLowDur time.Duration
	fastDur    time.Duration
	fastestDur time.Duration
}

func NewGasStation(gasStationURL string, updateDur time.Duration) (GasStation, error) {
	gs := &gasStation{
		baseURL:  gasStationURL,
		statsMux: new(sync.RWMutex),

		safeLowGas: ethfw.ToWei(0),
		fastGas:    ethfw.ToWei(0),
		fastestGas: ethfw.ToWei(0),
	}
	if err := gs.updateMetrics(); err != nil {
		return nil, err
	}
	go func() {
		t := time.NewTicker(updateDur)
		for range t.C {
			gs.updateMetrics()
		}
	}()
	return gs, nil
}

func (gs *gasStation) Estimate(priority GasPriority) (*ethfw.Wei, time.Duration) {
	switch priority {
	case GasPrioritySafeLow:
		gs.statsMux.RLock()
		gas, dur := gs.safeLowGas, gs.safeLowDur
		gs.statsMux.RUnlock()
		return gas, dur
	case GasPriorityFast:
		gs.statsMux.RLock()
		gas, dur := gs.fastGas, gs.fastDur
		gs.statsMux.RUnlock()
		if gas.Gwei() == 0 {
			return gs.Estimate(GasPrioritySafeLow)
		}
		return gas, dur
	case GasPriorityFastest:
		gs.statsMux.RLock()
		gas, dur := gs.fastestGas, gs.fastestDur
		gs.statsMux.RUnlock()
		if gas.Gwei() == 0 {
			return gs.Estimate(GasPriorityFast)
		}
		return gas, dur
	default:
		return ethfw.ToWei(0), 0
	}
}

func (gs *gasStation) updateMetrics() error {
	gasResp, err := gs.getGasResponse()
	if err != nil {
		return err
	}
	gs.statsMux.Lock()
	gs.blockNum = gasResp.BlockNum
	gs.safeLowGas = ethfw.Gwei(uint64(gasResp.SafeLow)).Div(10)
	gs.fastGas = ethfw.Gwei(uint64(gasResp.Fast)).Div(10)
	gs.fastestGas = ethfw.Gwei(uint64(gasResp.Fastest)).Div(10)
	gs.safeLowDur = time.Duration(gasResp.SafeLowWait * float64(time.Minute))
	gs.fastDur = time.Duration(gasResp.FastWait * float64(time.Minute))
	gs.fastestDur = time.Duration(gasResp.FastestWait * float64(time.Minute))
	gs.statsMux.Unlock()
	return nil
}

func (gs *gasStation) getGasResponse() (*ethGasResponse, error) {
	resp, err := http.Get(gs.baseURL)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("error %d: %s", resp.StatusCode, data)
		return nil, err
	}
	var ethGas *ethGasResponse
	if err := json.Unmarshal(data, &ethGas); err != nil {
		return nil, fmt.Errorf("response unmarshal error: %+v", err)
	} else if ethGas.AvgWait == 0 {
		return nil, fmt.Errorf("response is incomplete: %+v", *resp)
	}
	return ethGas, nil
}

type ethGasResponse struct {
	Average     float64 `json:"average"`
	AvgWait     float64 `json:"avgWait"`
	Fast        float64 `json:"fast"`
	Fastest     float64 `json:"fastest"`
	FastestWait float64 `json:"fastestWait"`
	FastWait    float64 `json:"fastWait"`
	SafeLow     float64 `json:"safeLow"`
	SafeLowWait float64 `json:"safeLowWait"`
	Speed       float64 `json:"speed"`
	BlockNum    uint64  `json:"blockNum"`
	BlockTime   float64 `json:"block_time"`
}
