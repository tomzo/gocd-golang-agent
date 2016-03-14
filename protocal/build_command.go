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

package protocal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type CommandTest struct {
	Command     *BuildCommand
	Expectation bool
}

type BuildCommand struct {
	Name             string
	Args             map[string]string
	RunIfConfig      string
	SubCommands      []*BuildCommand
	WorkingDirectory string
	Test             *CommandTest
}

func NewBuildCommand(name string) *BuildCommand {
	return &BuildCommand{
		Name:        name,
		RunIfConfig: "passed",
	}
}

func StartCommand(args map[string]string) *BuildCommand {
	return NewBuildCommand("start").SetArgs(args).RunIf("any")
}

func ComposeCommand(commands ...*BuildCommand) *BuildCommand {
	return NewBuildCommand("compose").AddCommands(commands...)
}

func EchoCommand(contents ...string) *BuildCommand {
	return NewBuildCommand("echo").SetArgs(listMap(contents...))
}

func EndCommand() *BuildCommand {
	return NewBuildCommand("end").RunIf("any")
}

func ReportCurrentStatusCommand(jobState string) *BuildCommand {
	args := map[string]string{"jobState": jobState}
	return NewBuildCommand("reportCurrentStatus").SetArgs(args)
}

func Parse(command map[string]interface{}) *BuildCommand {
	var cmd BuildCommand
	str, _ := json.Marshal(command)
	json.Unmarshal(str, &cmd)
	return &cmd
}

func (cmd *BuildCommand) AddCommands(commands ...*BuildCommand) *BuildCommand {
	cmd.SubCommands = append(cmd.SubCommands, commands...)
	return cmd
}

func (cmd *BuildCommand) SetArgs(args map[string]string) *BuildCommand {
	cmd.Args = args
	return cmd
}

func (cmd *BuildCommand) RunIf(c string) *BuildCommand {
	cmd.RunIfConfig = c
	return cmd
}

func (cmd *BuildCommand) ExtractArgList(size int) []string {
	ret := make([]string, size)
	for i := 0; i < size; i++ {
		ret[i] = cmd.Args[strconv.Itoa(i)]
	}
	return ret
}

func (cmd *BuildCommand) Dump(indent, step int) string {
	var buffer bytes.Buffer
	buffer.WriteString(strings.Repeat(" ", indent))
	buffer.WriteString(cmd.Name)
	for key, value := range cmd.Args {
		buffer.WriteString(fmt.Sprintf(" %v='%v'", key, value))
	}
	if "passed" != cmd.RunIfConfig {
		buffer.WriteString(fmt.Sprintf(" runIf:%v", cmd.RunIfConfig))
	}
	for _, subCmd := range cmd.SubCommands {
		buffer.WriteString("\n")
		buffer.WriteString(subCmd.Dump(indent+step, step))
	}
	return buffer.String()
}

func listMap(list ...string) map[string]string {
	ret := make(map[string]string, len(list))
	for i, s := range list {
		ret[strconv.Itoa(i)] = s
	}
	return ret
}
