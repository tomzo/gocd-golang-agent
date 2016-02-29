package libgogoagent

import "sync"

var state = map[string]string{
	"runtimeStatus": "Idle",
}

var lock sync.Mutex

func SetState(key, value string) {
	lock.Lock()
	defer lock.Unlock()
	state[key] = value
}

func GetState(key string) string {
	lock.Lock()
	defer lock.Unlock()
	return state[key]
}
