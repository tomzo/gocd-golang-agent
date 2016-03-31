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
	"github.com/gocd-contrib/gocd-golang-agent/stream"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

type BuildConsole struct {
	Url        *url.URL
	HttpClient *http.Client
	buffer     *bytes.Buffer
	stop       chan bool
	closed     chan bool
	write      chan []byte
}

func timestampPrefix() []byte {
	ts := time.Now().Format("15:04:05.000 ")
	return []byte(ts)
}

func MakeBuildConsole(httpClient *http.Client, url *url.URL) *BuildConsole {
	console := BuildConsole{
		HttpClient: httpClient,
		Url:        url,
		buffer:     bytes.NewBuffer(make([]byte, 0, 10*1024)),

		stop:   make(chan bool),
		closed: make(chan bool),
		write:  make(chan []byte),
	}
	go func() {
		defer func() {
			close(console.closed)
			LogInfo("build console closed")
		}()
		tw := stream.NewPrefixWriter(console.buffer, timestampPrefix)
		flushTick := time.NewTicker(5 * time.Second)
		defer flushTick.Stop()
		for {
			select {
			case log := <-console.write:
				tw.Write(log)
			case <-console.stop:
				console.Flush()
				return
			case <-flushTick.C:
				console.Flush()
			}
		}
	}()

	return &console
}

func (console *BuildConsole) Close() error {
	return closeAndWait(console.stop, console.closed, CancelCommandTimeout)
}

func (console *BuildConsole) Write(data []byte) (int, error) {
	console.write <- data
	return len(data), nil
}

func (console *BuildConsole) Flush() {
	if console.buffer.Len() == 0 {
		return
	}
	LogDebug("ConsoleLog: \n%v", console.buffer.String())

	req := http.Request{
		Method:        http.MethodPut,
		URL:           console.Url,
		Body:          ioutil.NopCloser(console.buffer),
		ContentLength: int64(console.buffer.Len()),
		Close:         true,
	}
	_, err := console.HttpClient.Do(&req)
	if err != nil {
		logger.Error.Printf("build console flush failed: %v", err)
	}
	console.buffer.Reset()
}
