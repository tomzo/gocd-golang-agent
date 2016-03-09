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

package libgocdgolangagent

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
	Lock       sync.Mutex
	Stop       chan int
}

func MakeBuildConsole(httpClient *http.Client, uri string, stop chan int) *BuildConsole {
	u, _ := url.Parse(uri + "&agentId=" + UUID)
	console := BuildConsole{
		HttpClient: httpClient,
		Url:        u,
		Buffer:     bytes.NewBuffer(make([]byte, 0, 10*1024)),
		Stop:       stop,
	}

	go func() {
		flushTick := time.NewTicker(5 * time.Second)
		defer flushTick.Stop()
		for {
			select {
			case <-console.Stop:
				console.Flush()
				LogInfo("build console closed")
				return
			case <-flushTick.C:
				console.Flush()
			}
		}
	}()

	return &console
}

func (console *BuildConsole) Write(data []byte) (int, error) {
	LogDebug("BuildConsole: %v", string(data))
	console.Lock.Lock()
	defer console.Lock.Unlock()
	return console.Buffer.Write(data)
}

func (console *BuildConsole) WriteLn(format string, a ...interface{}) {
	ln := fmt.Sprintf(format, a...)
	console.Write([]byte(fmt.Sprintf("%v %v\n", time.Now().Format("15:04:05.000"), ln)))
}

func (console *BuildConsole) Flush() {
	console.Lock.Lock()
	defer console.Lock.Unlock()
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
