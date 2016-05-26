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
package agent_test

import (
	"github.com/bmatcuk/doublestar"
	. "github.com/gocd-contrib/gocd-golang-agent/agent"
	"github.com/gocd-contrib/gocd-golang-agent/protocol"
	"github.com/xli/assert"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

var (
	echo = protocol.EchoCommand
)

func TestEcho(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocol.EchoCommand("echo hello world"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	assert.Equal(t, "echo hello world\n", trimTimestamp(log))
}

func TestExport(t *testing.T) {
	setUp(t)
	defer tearDown()

	os.Setenv("TEST_EXPORT", "EXPORT_VALUE")
	defer os.Setenv("TEST_EXPORT", "")
	os.Setenv("NO_EXPORT", "exec command should not have this")
	defer os.Setenv("NO_EXPORT", "")

	goServer.SendBuild(AgentId, buildId,
		protocol.ExportCommand("env1", "value1", "false"),
		protocol.ExportCommand("env2", "value2", "true"),
		protocol.ExportCommand("env1", "value4", "false"),
		protocol.ExportCommand("env2", "value5", "true"),
		protocol.ExportCommand("env2", "value6", "false"),
		protocol.ExportCommand("env2", "value6", ""),
		protocol.ExportCommand("env2", "", ""),
		protocol.ExportCommand("TEST_EXPORT"),
		protocol.ExecCommand("bash", "-c", "echo $env1"),
		protocol.ExecCommand("bash", "-c", "echo $NO_EXPORT"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := `setting environment variable 'env1' to value 'value1'
setting environment variable 'env2' to value '********'
overriding environment variable 'env1' with value 'value4'
overriding environment variable 'env2' with value '********'
overriding environment variable 'env2' with value 'value6'
overriding environment variable 'env2' with value 'value6'
overriding environment variable 'env2' with value ''
setting environment variable 'TEST_EXPORT' to value 'EXPORT_VALUE'
value4
`
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestExecCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId, protocol.ExecCommand("echo", "abcd"))

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	assert.Equal(t, "abcd\n", trimTimestamp(log))
}
func TestMkdirCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	wd := pipelineDir()
	goServer.SendBuild(AgentId, buildId,
		protocol.MkdirsCommand(relativePath(wd)),
		protocol.MkdirsCommand("path/in/pipeline/dir").Setwd(relativePath(wd)),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())
	_, err := os.Stat(filepath.Join(wd, "path/in/pipeline/dir"))
	assert.Nil(t, err)
}

func TestCleandirCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	wd := createTestProjectInPipelineDir()
	goServer.SendBuild(AgentId, buildId,
		protocol.CleandirCommand("test", "world2").Setwd(relativePath(wd)),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	matches, err := doublestar.Glob(filepath.Join(wd, "**/*.txt"))
	assert.Nil(t, err)
	sort.Strings(matches)
	expected := []string{
		"0.txt",
		"src/1.txt",
		"src/2.txt",
		"src/hello/3.txt",
		"src/hello/4.txt",
		"test/world2/10.txt",
		"test/world2/11.txt",
	}

	for i, f := range matches {
		actual := f[len(wd)+1:]
		assert.Equal(t, expected[i], actual)
	}
}

func TestFailCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId, protocol.FailCommand("something is wrong, please fail"))
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Failed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := Sprintf("ERROR: something is wrong, please fail\n")
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestSecretCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocol.SecretCommand("thisissecret", "$$$$$$"),
		protocol.SecretCommand("replacebydefaultmask"),
		protocol.EchoCommand("hello (thisissecret)"),
		protocol.EchoCommand("hello (replacebydefaultmask)"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := Sprintf("hello ($$$$$$)\nhello (********)\n")
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestShouldMaskSecretInExecOutput(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocol.SecretCommand("thisissecret", "$$$$$$"),
		protocol.ExecCommand("echo", "hello (thisissecret)"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := Sprintf("hello ($$$$$$)\n")
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestReplaceAgentBuildVairables(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocol.EchoCommand("hello ${agent.location}"),
		protocol.EchoCommand("hello ${agent.hostname}"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	config := GetConfig()
	expected := Sprintf("hello %v\nhello %v\n", config.WorkingDir, config.Hostname)
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestReplaceDateBuildVairables(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId, protocol.EchoCommand("${date}"))
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	log = strings.TrimSpace(trimTimestamp(log))
	_, err = time.Parse("2006-01-02 15:04:05 PDT", log)
	assert.Nil(t, err)
}

func TestFailBuildWhenThereIsUnsupportedBuildCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	cmd := protocol.NewBuildCommand("fancy")
	goServer.SendBuild(AgentId, buildId, cmd)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Failed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)

	expected := Sprintf("ERROR: Unknown build command: fancy\n")
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestShouldFailBuildIfWorkingDirIsSetToOutsideOfAgentWorkingDir(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		echo("echo hello world").Setwd("../../../"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Failed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())
}

func TestShouldFailBuildIfWorkingDirDoesNotExist(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		echo("echo hello world").Setwd("notexist/subdir"),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Failed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)

	config := GetConfig()
	expected := Sprintf("ERROR: Working directory \"%v/%v\" is not a directory\n", config.WorkingDir, "notexist/subdir")
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestReportStatusAndCompleting(t *testing.T) {
	setUp(t)
	defer tearDown()

	goServer.SendBuild(AgentId, buildId,
		protocol.ReportCurrentStatusCommand("Preparing"),
		protocol.ReportCurrentStatusCommand("Building"),
		protocol.ReportCompletingCommand(),
	)

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.NotEqual(t, "", GetState("cookie"))

	assert.Equal(t, "build Preparing", stateLog.Next())
	assert.Equal(t, "build Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())

	assert.Equal(t, "agent Idle", stateLog.Next())
}

func TestTestCommand(t *testing.T) {
	setUp(t)
	defer tearDown()

	wd := createTestProjectInPipelineDir()
	file := "src/hello/3.txt"
	dir := "src/hello"
	var tests = []struct {
		echo     string
		testArgs []string
		expected string
	}{
		{"file exist", []string{"-f", file}, "file exist\n"},
		{"file not exist", []string{"-f", file + "no"}, ""},
		{"dir is not file", []string{"-f", dir}, ""},
		{"file exist", []string{"-nf", file}, ""},
		{"file not exist", []string{"-nf", file + "no"}, "file not exist\n"},
		{"dir is not file", []string{"-nf", dir}, "dir is not file\n"},

		{"dir exist", []string{"-d", dir}, "dir exist\n"},
		{"dir exist", []string{"-nd", dir}, ""},
		{"dir not exist", []string{"-d", dir + "no"}, ""},
		{"dir not exist", []string{"-nd", dir + "no"}, "dir not exist\n"},
		{"file is not dir", []string{"-d", file}, ""},
		{"file is not dir", []string{"-nd", file}, "file is not dir\n"},

		{"equal", []string{"-eq", "hello", "echo", "hello"}, "equal\n"},
		{"not equal", []string{"-eq", "hello", "echo", "world"}, ""},
		{"equal", []string{"-neq", "hello", "echo", "hello"}, ""},
		{"not equal", []string{"-neq", "hello", "echo", "world"}, "not equal\n"},
	}

	for _, test := range tests {
		testCmd := protocol.TestCommand(test.testArgs...).Setwd(relativePath(wd))
		goServer.SendBuild(AgentId, buildId, protocol.CondCommand(testCmd, echo(test.echo)))
		assert.Equal(t, "agent Building", stateLog.Next())
		assert.Equal(t, "build Passed", stateLog.Next())
		assert.Equal(t, "agent Idle", stateLog.Next())
		log, err := goServer.ConsoleLog(buildId)
		if err != nil {
			t.Errorf("Can't find console log when test: %+v", test)
		}
		actual := trimTimestamp(log)
		if test.expected != actual {
			t.Errorf("test: %+v\nbut was '%v'", test, actual)
		}
		os.Truncate(goServer.ConsoleLogFile(buildId), 0)
	}
}

func TestTestCommandEchoShouldAlsoBeMaskedForSecrets(t *testing.T) {
	setUp(t)
	defer tearDown()

	testCmd := protocol.TestCommand("-eq", "$$$", "echo", "secret")
	goServer.SendBuild(AgentId, buildId,
		protocol.SecretCommand("secret", "$$$"),
		protocol.EchoCommand("hello world").SetTest(testCmd),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := Sprintf("hello world\n")
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestGenerateTestReportFromTestSuiteReports(t *testing.T) {
	setUp(t)
	defer tearDown()
	wd := createTestProjectInPipelineDir()
	copyTestReports(filepath.Join(wd, "reports"), "junit_report1.xml")
	copyTestReports(filepath.Join(wd, "reports"), "junit_report2.xml")

	goServer.SendBuild(AgentId, buildId,
		protocol.GenerateTestReportCommand("testoutput", "reports/junit_report1.xml", "reports/junit_report2.xml").Setwd(relativePath(wd)),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	reportPath := goServer.ArtifactFile(buildId, "testoutput/index.html")
	content, err := ioutil.ReadFile(reportPath)
	assert.Nil(t, err)
	assert.True(t, strings.Contains(string(content), "junit.framework.AssertionFailedError:"), Sprintf("wrong unit test report? %s", content))
}

func TestGenerateTestReportFromTestSuitesReport(t *testing.T) {
	setUp(t)
	defer tearDown()
	wd := createTestProjectInPipelineDir()
	copyTestReports(filepath.Join(wd, "reports"), "junit_report3.xml")

	goServer.SendBuild(AgentId, buildId,
		protocol.GenerateTestReportCommand("testoutput", "reports/junit_report3.xml").Setwd(relativePath(wd)),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	reportPath := goServer.ArtifactFile(buildId, "testoutput/index.html")
	content, err := ioutil.ReadFile(reportPath)
	assert.Nil(t, err)
	assert.True(t, strings.Contains(string(content), "<span class=\"tests_total_count\">1</span>"), Sprintf("wrong unit test report? %s", content))
}

func TestDoNothingIfGenerateTestReportSrcsIsEmpty(t *testing.T) {
	setUp(t)
	defer tearDown()
	wd := createTestProjectInPipelineDir()
	copyTestReports(filepath.Join(wd, "reports"), "junit_report3.xml")

	goServer.SendBuild(AgentId, buildId,
		protocol.GenerateTestReportCommand("testoutput").Setwd(relativePath(wd)),
	)
	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	reportPath := goServer.ArtifactFile(buildId, "testoutput/index.html")
	_, err := os.Stat(reportPath)
	assert.NotNil(t, err)
}

func TestConditionalCommand(t *testing.T) {
	setUp(t)
	defer tearDown()
	verify(t, []TestRow{
		{protocol.CondCommand(
			protocol.ComposeCommand(), protocol.EchoCommand("foo")),
			"foo\n", "Passed"},
		{protocol.CondCommand(
			protocol.FailCommand(""), protocol.EchoCommand("foo")),
			"", "Passed"},
		{protocol.CondCommand(
			protocol.ComposeCommand(),
			protocol.EchoCommand("foo"),
			protocol.EchoCommand("bar")),
			"foo\n", "Passed"},
		{protocol.CondCommand(
			protocol.FailCommand(""),
			protocol.EchoCommand("foo"),
			protocol.EchoCommand("bar")),
			"bar\n", "Passed"},
		{protocol.CondCommand(
			protocol.FailCommand(""), protocol.EchoCommand("1"),
			protocol.FailCommand(""), protocol.EchoCommand("2"),
			protocol.ComposeCommand(), protocol.EchoCommand("3"),
			protocol.ComposeCommand(), protocol.EchoCommand("4"),
			protocol.EchoCommand("else")),
			"3\n", "Passed"},
		{protocol.CondCommand(
			protocol.ComposeCommand(), protocol.FailCommand("foo")),
			"ERROR: foo\n", "Failed"},
	})

}

func TestAndCommand(t *testing.T) {
	setUp(t)
	defer tearDown()
	truthy := protocol.ComposeCommand()
	falsy := protocol.FailCommand("")
	and := protocol.AndCommand

	verify(t, []TestRow{
		{and(), "", "Passed"},
		{and(truthy), "", "Passed"},
		{and(falsy), "ERROR: \n", "Failed"},
		{and(truthy, truthy, truthy), "", "Passed"},
		{and(truthy, falsy, truthy), "ERROR: \n", "Failed"},
	})
}

func TestOrCommand(t *testing.T) {
	setUp(t)
	defer tearDown()
	truthy := protocol.ComposeCommand()
	falsy := protocol.FailCommand("")
	or := protocol.OrCommand

	verify(t, []TestRow{
		{or(), "", "Passed"},
		{or(truthy), "", "Passed"},
		{or(falsy), "ERROR: \n", "Failed"},
		{or(truthy, truthy, truthy), "", "Passed"},
		{or(truthy, falsy, truthy), "", "Passed"},
		{or(falsy, falsy, falsy), "ERROR: \n", "Failed"}})
}

type TestRow struct {
	command  *protocol.BuildCommand
	expected string
	result   string
}

func verify(t *testing.T, testRows []TestRow) {
	for _, row := range testRows {
		goServer.SendBuild(AgentId, buildId, row.command)
		assert.Equal(t, "agent Building", stateLog.Next())
		assert.Equal(t, "build "+row.result, stateLog.Next())
		assert.Equal(t, "agent Idle", stateLog.Next())
		log, err := goServer.ConsoleLog(buildId)
		if err != nil && row.expected != "" {
			t.Errorf("Can't find console log when test: %+v, error: %+v", row, err)
		}
		actual := trimTimestamp(log)
		if row.expected != actual {
			t.Errorf("test: %+v\nbut was '%v'", row.expected, actual)
		}
		os.Truncate(goServer.ConsoleLogFile(buildId), 0)
	}
}

func copyTestReports(wd, rep string) {
	Mkdirs(wd)
	rep1 := filepath.Join(DIR(), "..", "junit", "test", rep)
	src, _ := os.Open(rep1)
	defer src.Close()
	dest, _ := os.Create(filepath.Join(wd, rep))
	defer dest.Close()

	_, err := io.Copy(dest, src)
	if err != nil {
		panic(err)
	}
}

func DIR() string {
	_, filename, _, _ := runtime.Caller(1)
	return filepath.Dir(filename)
}
