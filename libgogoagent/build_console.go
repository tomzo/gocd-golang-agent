package libgogoagent

import (
	"bytes"
	"fmt"
	"log"
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

func MakeBuildConsole(httpClient *http.Client, uri string) *BuildConsole {
	log.Println(uri)
	u, _ := url.Parse(uri + "&agentId=" + uuid)
	console := BuildConsole{
		HttpClient: httpClient,
		Url:        u,
		Buffer:     bytes.NewBuffer(make([]byte, 0, 10*1024)),
	}

	go func() {
		for {
			select {
			case <-console.Stop:
				console.Flush()
				return
			default:
				console.Flush()
				time.Sleep(5 * time.Second)
			}
		}
	}()

	return &console
}

func (console *BuildConsole) Write(str string) {
	console.Lock.Lock()
	defer console.Lock.Unlock()
	console.Buffer.Write([]byte(str))
}

func (console *BuildConsole) WriteLn(ln string) {
	console.Write(fmt.Sprintf("%v %v\n", time.Now().Format("15:04:05.000"), ln))
}

func (console *BuildConsole) Read(p []byte) (int, error) {
	return console.Buffer.Read(p)
}

func (console *BuildConsole) Close() error {
	console.Buffer.Reset()
	return nil
}

func (console *BuildConsole) Flush() {
	console.Lock.Lock()
	defer console.Lock.Unlock()
	req := http.Request{
		Method:        http.MethodPut,
		URL:           console.Url,
		Body:          console,
		ContentLength: int64(console.Buffer.Len()),
	}
	_, err := console.HttpClient.Do(&req)
	if err != nil {
		log.Println(err)
	}
}
