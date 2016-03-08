package libgocdgolangagent

import (
	"encoding/json"
)

type CommandTest struct {
	Command     *BuildCommand
	Expectation bool
}

type BuildCommand struct {
	Name, RunIfConfig, WorkingDirectory string
	Test                                *CommandTest
	Args                                []interface{}
	SubCommands                         []*BuildCommand
}

func MakeBuildCommand(command map[string]interface{}) *BuildCommand {
	var cmd BuildCommand
	str, _ := json.Marshal(command)
	json.Unmarshal(str, &cmd)
	return &cmd
}
