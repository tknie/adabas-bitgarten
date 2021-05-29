/*
* Copyright © 2018-2019 private, Darmstadt, Germany and/or its licensors
*
* SPDX-License-Identifier: Apache-2.0
*
*   Licensed under the Apache License, Version 2.0 (the "License");
*   you may not use this file except in compliance with the License.
*   You may obtain a copy of the License at
*
*       http://www.apache.org/licenses/LICENSE-2.0
*
*   Unless required by applicable law or agreed to in writing, software
*   distributed under the License is distributed on an "AS IS" BASIS,
*   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*   See the License for the specific language governing permissions and
*   limitations under the License.
*
 */

package store

import (
	"fmt"
	"sync"
)

// WorkerType worker function
type WorkerType func(job string)

// Worker worker function
type Worker struct {
	nrWorkers int
	Jobs      chan string
	End       chan bool
	Listener  WorkerType
	wg        sync.WaitGroup
}

// InitWorker init worker functions
func InitWorker(nrWorker int, l WorkerType) *Worker {
	wr := &Worker{nrWorkers: nrWorker,
		Jobs:     make(chan string, nrWorker),
		End:      make(chan bool, 1),
		Listener: l}
	fmt.Println("Init ", nrWorker)

	wr.wg.Add(nrWorker)
	for w := 1; w <= nrWorker; w++ {
		go wr.workerFunc(w)
	}
	return wr
}

func (wr *Worker) workerFunc(w int) {
	for {
		select {
		case p := <-wr.Jobs:
			wr.Listener(p)
		case <-wr.End:
			wr.wg.Done()
			return
		}
	}
}

// WaitEnd wait for end of worker
func (wr *Worker) WaitEnd() {
	for i := 0; i < wr.nrWorkers; i++ {
		wr.End <- true
	}
	wr.wg.Wait()
}
