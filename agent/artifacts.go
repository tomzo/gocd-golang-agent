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
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Artifacts struct {
	httpClient *http.Client
}

func (u *Artifacts) DownloadFile(source *url.URL, destPath string) (err error) {
	dir, _ := filepath.Split(destPath)
	err = Mkdirs(dir)
	if err != nil {
		return err
	}
	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	return u.downloadFile(source, destFile)
}

func (u *Artifacts) DownloadDir(source *url.URL, destPath string) error {
	zipfile, err := ioutil.TempFile("", "tmp.zip")
	if err != nil {
		return err
	}
	defer os.Remove(zipfile.Name())
	LogDebug("tmp file created for download zipped dir")
	err = u.downloadFile(source, zipfile)
	if err != nil {
		return err
	}

	zipReader, err := zip.OpenReader(zipfile.Name())
	if err != nil {
		return err
	}
	LogDebug("unzip to %v", destPath)
	defer zipReader.Close()
	destDir := filepath.Dir(destPath)
	for _, file := range zipReader.File {
		dest := filepath.Join(destDir, file.FileHeader.Name)
		if file.FileHeader.FileInfo().IsDir() {
			LogDebug("mkdirs %v", dest)
			err = Mkdirs(dest)
		} else {
			LogDebug("extract file %v => %v", file.FileHeader.Name, dest)
			err = u.extractFile(file, dest)
		}
		if err != nil {
			return err
		}
	}
	LogDebug("unzip finished")
	return nil
}

func (u *Artifacts) downloadFile(source *url.URL, destFile *os.File) (err error) {
	defer destFile.Close()
	LogDebug("download file %v => %v", source, destFile.Name())
	retry := 0
startDownload:
	resp, err := u.httpClient.Get(source.String())
	if err != nil {
		return
	}
	LogDebug("response: %v", resp.Status)
	if resp.StatusCode == http.StatusAccepted {
		LogDebug("Server responsed StatusAccepted, sleep 1 sec and start download again")
		time.Sleep(1 * time.Second)
		goto startDownload
	}
	if resp.StatusCode != http.StatusOK {
		if retry < 3 {
			retry++
			LogDebug("sleep %v sec and start download again", retry)
			time.Sleep(time.Duration(retry) * time.Second)
			goto startDownload
		} else {
			return Err("tried %v times to download [%v] and all failed.", retry, source)
		}
	}
	defer resp.Body.Close()
	_, err = io.Copy(destFile, resp.Body)
	return
}

func (u *Artifacts) VerifyChecksum(srcPath, destPath, checksumFname string) error {
	destInfo, err := os.Stat(destPath)
	if err != nil {
		return err
	}
	if destInfo.IsDir() {
		return filepath.Walk(destPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			srcFname := Join("/", srcPath, path[len(destPath)+1:])
			return u.VerifyChecksumFile(srcFname, path, checksumFname)
		})
	} else {
		return u.VerifyChecksumFile(srcPath, destPath, checksumFname)
	}
}

func (u *Artifacts) VerifyChecksumFile(srcFname, fname, checksumFname string) error {
	md5, err := ComputeMd5(fname)
	if err != nil {
		return err
	}
	checksum, err := ioutil.ReadFile(checksumFname)
	if err != nil {
		return err
	}
	properties := ParseChecksum(string(checksum))
	// Convert path used as key name in properties, because md5.checksum always has unix / slashes
	srcFname=filepath.ToSlash(srcFname)
	if properties[srcFname] == "" {
		return Err("[WARN] The md5checksum value of the artifact [%v] was not found on the server. Hence, Go could not verify the integrity of its contents.", srcFname)
	} else if properties[srcFname] != md5 {
		return Err("[ERROR] Verification of the integrity of the artifact [%v] failed. The artifact file on the server may have changed since its original upload.", srcFname)
	} else {
		return nil
	}
}

func (u *Artifacts) Upload(source, destPath string, destURL *url.URL) (err error) {
	zipped, checksum, err := u.zipSource(source, destPath)
	defer os.Remove(zipped)
	if err != nil {
		return
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	err = u.writeFilePart(writer, zipped, "zipfile")
	if err != nil {
		return
	}
	err = u.writePart(writer, bytes.NewBufferString(checksum), "file_checksum", "checksum_file")
	if err != nil {
		return
	}
	err = writer.Close()
	if err != nil {
		return
	}

	attempt := 1
tryPost:
	attemptUrl := AppendUrlParam(destURL, "attempt", strconv.Itoa(attempt))
	statusCode, err := u.post(source, writer.FormDataContentType(), attemptUrl, &body)
	// client side errors, no retry
	if err != nil {
		return
	}
	// success
	if statusCode == http.StatusCreated {
		return
	}
	// handle errors
	if statusCode == http.StatusRequestEntityTooLarge {
		info, _ := os.Stat(zipped)
		return Err("Artifact upload for file %s (Size: %d) was denied by the server. This usually happens when server runs out of disk space.", source, info.Size())
	}
	// retry for other errors
	if attempt < 3 {
		attempt++
		goto tryPost
	}
	return Err("Failed to upload %v. Server response: %v", source, statusCode)
}

func (u *Artifacts) post(source, contentType string, destURL *url.URL, body *bytes.Buffer) (statusCode int, err error) {
	req, err := http.NewRequest("POST", destURL.String(), body)
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", contentType)
	req.Header.Add("Confirm","true")

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return
	}
	return resp.StatusCode, nil
}

func (u *Artifacts) writeFilePart(writer *multipart.Writer, path, paramName string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return u.writePart(writer, file, paramName, filepath.Base(path))
}

func (u *Artifacts) writePart(writer *multipart.Writer, src io.Reader, fieldname, filename string) error {
	part, err := writer.CreateFormFile(fieldname, filename)
	if err != nil {
		return err
	}
	_, err = io.Copy(part, src)
	return err
}

func (u *Artifacts) zipSource(source string, dest string) (string, string, error) {
	zipfile, err := ioutil.TempFile("", "tmp.zip")
	if err != nil {
		return "", "", err
	}
	defer zipfile.Close()
	w := zip.NewWriter(zipfile)
	defer w.Close()

	var checksum bytes.Buffer
	checksum.WriteString(Sprintf("#\n#%v\n", time.Now()))
	err = filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		destFile := dest
		if path != source {
			// source is a directory, find relative path
			// from source and attach to dest path
			rel := path[len(source):]
			if strings.HasPrefix(rel, string(os.PathSeparator)) {
				rel = rel[1:]
			}
			if dest == "" {
				destFile = rel
			} else {
				destFile = dest + "/" + rel
			}
		}
		// Convert slash to Linux slash especally on Windows
		destFile=filepath.ToSlash(destFile)
		md5, err := ComputeMd5(path)
		if err != nil {
			return err
		}
		checksum.WriteString(Sprintf("%v=%v\n", destFile, md5))

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		writer, err := w.Create(destFile)
		if err != nil {
			return err
		}

		_, err = io.Copy(writer, file)
		return err
	})
	return zipfile.Name(), checksum.String(), err
}

func (u *Artifacts) extractFile(file *zip.File, dest string) error {
	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	err = os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return err
	}
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, rc)
	return err
}
