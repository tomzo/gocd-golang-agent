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
	"fmt"
	"strconv"
	"strings"
)

var (
	CommandCompose             = "compose"
	CommandExport              = "export"
	CommandTest                = "test"
	CommandExec                = "exec"
	CommandEcho                = "echo"
	CommandUploadArtifact      = "uploadArtifact"
	CommandReportCurrentStatus = "reportCurrentStatus"
	CommandReportCompleting    = "reportCompleting"
	CommandReportCompleted     = "reportCompleted"
	// todo
	CommandMkdirs   = "mkdirs"
	CommandCleandir = "cleandir"
	CommandFail     = "fail"
	CommandSecret   = "secret"
)

type BuildCommandTest struct {
	Command     *BuildCommand
	Expectation bool
}

type BuildCommand struct {
	Name             string
	Args             map[string]string
	RunIfConfig      string
	SubCommands      []*BuildCommand
	WorkingDirectory string
	Test             *BuildCommandTest
}

func NewBuildCommand(name string) *BuildCommand {
	return &BuildCommand{
		Name:        name,
		RunIfConfig: "passed",
	}
}

func NewBuild(id, locator, locatorForDisplay,
	consoleURL, artifactUploadBaseUrl, propertyBaseUrl string,
	commands ...*BuildCommand) *Build {
	return &Build{
		BuildId:                id,
		BuildLocator:           locator,
		BuildLocatorForDisplay: locator,
		ConsoleURI:             consoleURL,
		ArtifactUploadBaseUrl:  artifactUploadBaseUrl,
		PropertyBaseUrl:        propertyBaseUrl,
		BuildCommand:           ComposeCommand(commands...),
	}
}

func ComposeCommand(commands ...*BuildCommand) *BuildCommand {
	return NewBuildCommand(CommandCompose).AddCommands(commands...)
}

func EchoCommand(contents ...string) *BuildCommand {
	return NewBuildCommand(CommandEcho).SetArgs(listMap(contents...))
}

func ExecCommand(args ...string) *BuildCommand {
	return NewBuildCommand(CommandExec).SetArgs(listMap(args...))
}

func ExportCommand(kvs ...string) *BuildCommand {
	args := map[string]string{"name": kvs[0]}
	if len(kvs) == 3 {
		args["value"] = kvs[1]
		args["secure"] = kvs[2]
	}
	return NewBuildCommand(CommandExport).SetArgs(args)
}

func ReportCurrentStatusCommand(jobState string) *BuildCommand {
	args := map[string]string{"jobState": jobState}
	return NewBuildCommand(CommandReportCurrentStatus).SetArgs(args)
}

func ReportCompletingCommand() *BuildCommand {
	return NewBuildCommand(CommandReportCompleting).RunIf("any")
}

func ReportCompletedCommand() *BuildCommand {
	return NewBuildCommand(CommandReportCompleted).RunIf("any")
}

func TestCommand(flag, path string) *BuildCommand {
	args := map[string]string{
		"flag": flag,
		"path": path,
	}
	return NewBuildCommand(CommandTest).SetArgs(args)
}

func MkdirsCommand(path string) *BuildCommand {
	args := map[string]string{"path": path}
	return NewBuildCommand(CommandMkdirs).SetArgs(args)
}

func UploadArtifactCommand(src, dest string) *BuildCommand {
	args := map[string]string{
		"src":  src,
		"dest": dest,
	}
	return NewBuildCommand(CommandUploadArtifact).SetArgs(args)
}

func (cmd *BuildCommand) AddCommands(commands ...*BuildCommand) *BuildCommand {
	cmd.SubCommands = append(cmd.SubCommands, commands...)
	return cmd
}

func (cmd *BuildCommand) SetArgs(args map[string]string) *BuildCommand {
	cmd.Args = args
	return cmd
}

func (cmd *BuildCommand) SetTest(test *BuildCommand) *BuildCommand {
	cmd.Test = &BuildCommandTest{
		Command:     test,
		Expectation: true,
	}
	return cmd
}

func (cmd *BuildCommand) Setwd(wd string) *BuildCommand {
	cmd.WorkingDirectory = wd
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

func (cmd *BuildCommand) String() string {
	return cmd.Dump(2, 2)
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
