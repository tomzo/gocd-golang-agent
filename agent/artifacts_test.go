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
	"bytes"
	. "github.com/gocd-contrib/gocd-golang-agent/agent"
	"github.com/gocd-contrib/gocd-golang-agent/protocal"
	"github.com/xli/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadArtifactFile(t *testing.T) {
	setUp(t)
	defer tearDown()

	artifactWd := createPipelineDir()
	fname := createTestFile(artifactWd, "file.txt")

	goServer.SendBuild(AgentId, buildId,
		protocal.UploadArtifactCommand(fname, "").Setwd(artifactWd))

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := sprintf("Uploading artifacts from %v/%v to [defaultRoot]\n", artifactWd, fname)
	assert.Equal(t, expected, trimTimestamp(log))

	f := goServer.ArtifactFile(buildId, fname)
	finfo, err := os.Stat(f)
	assert.Nil(t, err)
	assert.Equal(t, fname, finfo.Name())

	content, err := ioutil.ReadFile(f)
	assert.Nil(t, err)
	assert.Equal(t, "file created for test", string(content))

	checksum, err := goServer.Checksum(buildId)
	assert.Nil(t, err)
	assert.Equal(t, fname+"=41e43efb30d3fbfcea93542157809ac0\n", filterComments(checksum))
}

func TestUploadArtifactFailed(t *testing.T) {
	setUp(t)
	defer tearDown()

	artifactWd := createPipelineDir()
	fname := "nofile"

	goServer.SendBuild(AgentId, buildId,
		protocal.UploadArtifactCommand(fname, "").Setwd(artifactWd),
		protocal.ReportCompletedCommand(),
	)

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Failed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := sprintf("stat %v/%v: no such file or directory\n",
		artifactWd, fname)
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestUploadDirectory(t *testing.T) {
	setUp(t)
	defer tearDown()

	wd := createTestProjectInPipelineDir()
	dir := "src"
	goServer.SendBuild(AgentId, buildId,
		protocal.UploadArtifactCommand(dir, "").Setwd(wd))

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	expected := sprintf("Uploading artifacts from %v/%v to [defaultRoot]\n", wd, dir)
	assert.Equal(t, expected, trimTimestamp(log))

	checksum, err := goServer.Checksum(buildId)
	assert.Nil(t, err)
	expectedChecksum := `1.txt=41e43efb30d3fbfcea93542157809ac0
2.txt=41e43efb30d3fbfcea93542157809ac0
hello/3.txt=41e43efb30d3fbfcea93542157809ac0
hello/4.txt=41e43efb30d3fbfcea93542157809ac0
`
	assert.Equal(t, expectedChecksum, filterComments(checksum))

	uploadedDir := goServer.ArtifactFile(buildId, "")
	count := 0
	err = filepath.Walk(uploadedDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		content, err := ioutil.ReadFile(path)
		assert.Nil(t, err)
		assert.Equal(t, "file created for test", string(content))
		count++
		return nil
	})
	assert.Nil(t, err)
	assert.Equal(t, 4, count)
}

func filterComments(str string) string {
	var ret bytes.Buffer
	for _, l := range split(str, "\n") {
		if startWith(l, "#") || l == "" {
			continue
		}
		ret.WriteString(l)
		ret.WriteString("\n")
	}
	return ret.String()
}
