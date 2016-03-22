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
	"sort"
	"testing"
)

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

func TestUploadArtifactFailedWhenServerHasNotEnoughDiskspace(t *testing.T) {
	setUp(t)
	defer tearDown()
	goServer.SetMaxRequestEntitySize(1000)
	defer goServer.SetMaxRequestEntitySize(0)

	wd := createTestProjectInPipelineDir()
	var buf bytes.Buffer
	for i := 0; i < 10000; i++ {
		buf.WriteString("large file content")
	}
	writeFile(wd, "large.txt", buf.String())
	goServer.SendBuild(AgentId, buildId,
		protocal.UploadArtifactCommand("large.txt", "").Setwd(wd),
		protocal.ReportCompletedCommand())

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Failed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)
	f := `Uploading artifacts from %v/large.txt to [defaultRoot]
Artifact upload for file %v/large.txt (Size: 609) was denied by the server. This usually happens when server runs out of disk space.
`
	expected := sprintf(f, wd, wd)
	assert.Equal(t, expected, trimTimestamp(log))
}

func TestUploadDirectory1(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "src", "",
		`src/1.txt=41e43efb30d3fbfcea93542157809ac0
src/2.txt=41e43efb30d3fbfcea93542157809ac0
src/hello/3.txt=41e43efb30d3fbfcea93542157809ac0
src/hello/4.txt=41e43efb30d3fbfcea93542157809ac0
`,
		map[string]string{
			"src": "[defaultRoot]",
		})
}

func TestUploadDirectory2(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "src/hello", "",
		`hello/3.txt=41e43efb30d3fbfcea93542157809ac0
hello/4.txt=41e43efb30d3fbfcea93542157809ac0
`,
		map[string]string{
			"src/hello": "[defaultRoot]",
		})
}

func TestUploadDirectory3(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "src/hello", "dest",
		`dest/hello/3.txt=41e43efb30d3fbfcea93542157809ac0
dest/hello/4.txt=41e43efb30d3fbfcea93542157809ac0
`,
		map[string]string{
			"src/hello": "dest",
		})
}

func TestUploadFile1(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "src/hello/4.txt", "",
		"4.txt=41e43efb30d3fbfcea93542157809ac0\n",
		map[string]string{
			"src/hello/4.txt": "[defaultRoot]"})
}

func TestUploadFile2(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "src/hello/4.txt", "dest/subdir",
		"dest/subdir/4.txt=41e43efb30d3fbfcea93542157809ac0\n",
		map[string]string{
			"src/hello/4.txt": "dest/subdir"})
}

func TestUploadMatchedFiles1(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "src/hello/*.txt", "dest",
		`dest/3.txt=41e43efb30d3fbfcea93542157809ac0
dest/4.txt=41e43efb30d3fbfcea93542157809ac0
`,
		map[string]string{
			"src/hello/3.txt": "dest",
			"src/hello/4.txt": "dest"})
}

func TestUploadMatchedFiles2(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "src/**/3.txt", "dest",
		`dest/hello/3.txt=41e43efb30d3fbfcea93542157809ac0
`,
		map[string]string{
			"src/hello/3.txt": "dest/hello"})
}

func TestUploadMatchedFiles3(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "test/w*/10.txt", "dest",
		`dest/world/10.txt=41e43efb30d3fbfcea93542157809ac0
dest/world2/10.txt=41e43efb30d3fbfcea93542157809ac0
`,
		map[string]string{
			"test/world/10.txt":  "dest/world",
			"test/world2/10.txt": "dest/world2"})
}

func TestUploadMatchedFiles4(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "src/**/*.txt", "dest",
		`dest/1.txt=41e43efb30d3fbfcea93542157809ac0
dest/2.txt=41e43efb30d3fbfcea93542157809ac0
dest/hello/3.txt=41e43efb30d3fbfcea93542157809ac0
dest/hello/4.txt=41e43efb30d3fbfcea93542157809ac0
`,
		map[string]string{
			"src/1.txt":       "dest",
			"src/2.txt":       "dest",
			"src/hello/3.txt": "dest/hello",
			"src/hello/4.txt": "dest/hello"})
}

func TestUploadMatchedFiles5(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "**/*.txt", "dest",
		`dest/0.txt=41e43efb30d3fbfcea93542157809ac0
dest/src/1.txt=41e43efb30d3fbfcea93542157809ac0
dest/src/2.txt=41e43efb30d3fbfcea93542157809ac0
dest/src/hello/3.txt=41e43efb30d3fbfcea93542157809ac0
dest/src/hello/4.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/5.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/6.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/7.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/world/10.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/world/11.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/world/8.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/world/9.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/world2/10.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/world2/11.txt=41e43efb30d3fbfcea93542157809ac0
`,
		map[string]string{
			"0.txt":              "dest",
			"src/1.txt":          "dest/src",
			"src/2.txt":          "dest/src",
			"src/hello/3.txt":    "dest/src/hello",
			"src/hello/4.txt":    "dest/src/hello",
			"test/5.txt":         "dest/test",
			"test/6.txt":         "dest/test",
			"test/7.txt":         "dest/test",
			"test/world/8.txt":   "dest/test/world",
			"test/world/9.txt":   "dest/test/world",
			"test/world/10.txt":  "dest/test/world",
			"test/world/11.txt":  "dest/test/world",
			"test/world2/10.txt": "dest/test/world2",
			"test/world2/11.txt": "dest/test/world2",
		})
}

func TestUploadMatchedFiles6(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "*/*.txt", "dest",
		`dest/src/1.txt=41e43efb30d3fbfcea93542157809ac0
dest/src/2.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/5.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/6.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/7.txt=41e43efb30d3fbfcea93542157809ac0
`,
		map[string]string{
			"src/1.txt":  "dest/src",
			"src/2.txt":  "dest/src",
			"test/5.txt": "dest/test",
			"test/6.txt": "dest/test",
			"test/7.txt": "dest/test"})
}

func TestUploadMatchedFiles7(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "*/world/?.txt", "dest",
		`dest/test/world/8.txt=41e43efb30d3fbfcea93542157809ac0
dest/test/world/9.txt=41e43efb30d3fbfcea93542157809ac0
`,
		map[string]string{
			"test/world/8.txt": "dest/test/world",
			"test/world/9.txt": "dest/test/world"})
}

func TestUploadDestIsWindowPathFormat(t *testing.T) {
	setUp(t)
	defer tearDown()
	testUpload(t, "test/world/10.txt", "dest\\dir",
		`dest/dir/10.txt=41e43efb30d3fbfcea93542157809ac0
`,
		map[string]string{"test/world/10.txt": "dest/dir"})

}

func TestProcessMultipleUploadArtifactCommands(t *testing.T) {
	setUp(t)
	defer tearDown()

	wd := createTestProjectInPipelineDir()
	goServer.SendBuild(AgentId, buildId,
		protocal.UploadArtifactCommand("src/hello/3.txt", "dest").Setwd(wd),
		protocal.UploadArtifactCommand("test/5.txt", "").Setwd(wd),
		protocal.UploadArtifactCommand("test/**/10.txt", "dest").Setwd(wd),
		protocal.ReportCompletedCommand())

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	assertConsoleLog(t, wd, map[string]string{
		"src/hello/3.txt":    "dest",
		"test/5.txt":         "[defaultRoot]",
		"test/world/10.txt":  "dest/world",
		"test/world2/10.txt": "dest/world2",
	})

	uploadedChecksum, err := goServer.Checksum(buildId)
	assert.Nil(t, err)
	checksum := `dest/3.txt=41e43efb30d3fbfcea93542157809ac0
5.txt=41e43efb30d3fbfcea93542157809ac0
dest/world/10.txt=41e43efb30d3fbfcea93542157809ac0
dest/world2/10.txt=41e43efb30d3fbfcea93542157809ac0
`
	assert.Equal(t, checksum, filterComments(uploadedChecksum))
}

func testUpload(t *testing.T, srcPath, destDir, checksum string, src2dest map[string]string) {
	wd := createTestProjectInPipelineDir()
	goServer.SendBuild(AgentId, buildId,
		protocal.UploadArtifactCommand(srcPath, destDir).Setwd(wd),
		protocal.ReportCompletedCommand())

	assert.Equal(t, "agent Building", stateLog.Next())
	assert.Equal(t, "build Passed", stateLog.Next())
	assert.Equal(t, "agent Idle", stateLog.Next())

	assertConsoleLog(t, wd, src2dest)

	uploadedChecksum, err := goServer.Checksum(buildId)
	assert.Nil(t, err)
	assert.Equal(t, checksum, filterComments(uploadedChecksum))

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
	assert.Equal(t, len(split(checksum, "\n"))-1, count)
}

func assertConsoleLog(t *testing.T, wd string, src2dest map[string]string) {
	log, err := goServer.ConsoleLog(buildId)
	assert.Nil(t, err)

	expected := make([]string, len(src2dest)+1)
	i := 0
	for src, dest := range src2dest {
		expected[i] = sprintf("Uploading artifacts from %v/%v to %v", wd, src, dest)
		i++
	}
	actual := split(trimTimestamp(log), "\n")
	sort.Strings(expected)
	sort.Strings(actual)
	assert.Equal(t, Join("\n", expected...), Join("\n", actual...))
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
