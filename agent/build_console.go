/*
 * Copyright 2016 ThoughtWorks, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agent

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type BuildConsole struct {
	Url        *url.URL
	HttpClient *http.Client
	Buffer     *bytes.Buffer
	stop       chan bool
	closed     *sync.WaitGroup
	write      chan []byte
}

func MakeBuildConsole(agentId string, httpClient *http.Client, uri string) *BuildConsole {
	u, _ := url.Parse(uri + "&agentId=" + agentId)
	console := BuildConsole{
		HttpClient: httpClient,
		Url:        u,
		Buffer:     bytes.NewBuffer(make([]byte, 0, 10*1024)),
		stop:       make(chan bool),
		closed:     &sync.WaitGroup{},
		write:      make(chan []byte),
	}
	console.closed.Add(1)
	go func() {
		flushTick := time.NewTicker(5 * time.Second)
		defer flushTick.Stop()
		for {
			select {
			case data := <-console.write:
				LogDebug("BuildConsole: %v", string(data))
				console.Buffer.Write(data)
			case <-console.stop:
				console.Flush()
				LogInfo("build console closed")
				console.closed.Done()
				return
			case <-flushTick.C:
				console.Flush()
			}
		}
	}()

	return &console
}

func (console *BuildConsole) Close() {
	console.stop <- true
	console.closed.Wait()
}

func (console *BuildConsole) Write(data []byte) (int, error) {
	console.write <- data
	return len(data), nil
}

func (console *BuildConsole) WriteLn(format string, a ...interface{}) {
	ln := fmt.Sprintf(format, a...)
	console.Write([]byte(fmt.Sprintf("%v %v\n", time.Now().Format("15:04:05.000"), ln)))
}

func (console *BuildConsole) Flush() {
	LogDebug("build console flush, buffer len: %v", console.Buffer.Len())
	if console.Buffer.Len() == 0 {
		return
	}
	req := http.Request{
		Method:        http.MethodPut,
		URL:           console.Url,
		Body:          ioutil.NopCloser(console.Buffer),
		ContentLength: int64(console.Buffer.Len()),
		Close:         true,
	}
	_, err := console.HttpClient.Do(&req)
	if err != nil {
		logger.Error.Printf("build console flush failed: %v", err)
	}
	console.Buffer.Reset()
}
