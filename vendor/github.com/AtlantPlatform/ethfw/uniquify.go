// Copyright 2017, 2018 Tensigma Ltd. All rights reserved.
// Use of this source code is governed by Microsoft Reference Source
// License (MS-RSL) that can be found in the LICENSE file.

package ethfw

import "sync"

type Uniqify struct {
	mux   *sync.Mutex
	tasks map[string]*sync.WaitGroup
}

func NewUniqify() *Uniqify {
	return &Uniqify{
		mux:   new(sync.Mutex),
		tasks: make(map[string]*sync.WaitGroup),
	}
}

func (u *Uniqify) Call(id string, callable func() error) error {
	errC := make(chan error)

	func() {
		u.mux.Lock()
		defer u.mux.Unlock()
		prevWG := u.tasks[id]
		if prevWG != nil {
			go func() {
				prevWG.Wait()
				go func() {
					errC <- u.Call(id, callable)
				}()
			}()
		}
		wg := new(sync.WaitGroup)
		wg.Add(1)
		u.tasks[id] = wg
		go func() {
			defer wg.Done()
			errC <- callable()
			u.mux.Lock()
			delete(u.tasks, id)
			u.mux.Unlock()
		}()
	}()

	return <-errC
}
