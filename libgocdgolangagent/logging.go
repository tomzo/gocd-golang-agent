package libgocdgolangagent

import (
	"fmt"
	"os"
)

var debug = os.Getenv("DEBUG") != ""

func LogDebug(format string, v ...interface{}) {
	if debug {
		fmt.Printf("[DEBUG] "+format+"\n", v...)
	}
}

func LogInfo(format string, v ...interface{}) {
	fmt.Printf("[INFO] "+format+"\n", v...)
}
