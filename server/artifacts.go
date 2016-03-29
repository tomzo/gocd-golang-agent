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

package server

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func artifactsHandler(s *Server) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodPost:
			handleArtifactsUpload(s, w, req)
		case http.MethodGet:
			handleArtifactDownload(s, w, req)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}
}

func handleArtifactDownload(s *Server, w http.ResponseWriter, req *http.Request) {
	buildId := parseBuildId(req.URL.Path)
	file := req.URL.Query()["file"]
	if len(file) == 0 {
		w.WriteHeader(http.StatusNotAcceptable)
		return
	}
	fullPath := s.ArtifactFile(buildId, file[0])
	f, err := os.Open(fullPath)
	defer f.Close()
	if err != nil {
		s.responseBadRequest(err, w)
	} else {
		io.Copy(w, f)
	}
}

func handleArtifactsUpload(s *Server, w http.ResponseWriter, req *http.Request) {
	buildId := parseBuildId(req.URL.Path)
	form, err := req.MultipartReader()
	if err != nil {
		s.responseBadRequest(err, w)
		return
	}
	for {
		part, err := form.NextPart()
		if err == io.EOF {
			break
		}
		switch part.FormName() {
		case "zipfile":
			err = extractToArtifactDir(s, buildId, part)
			if err != nil {
				s.responseInternalError(err, w)
				return
			}
		case "file_checksum":
			bytes, err := ioutil.ReadAll(part)
			if err != nil {
				s.responseInternalError(err, w)
				return
			}
			err = s.appendToFile(s.ChecksumFile(buildId), bytes)
			if err != nil {
				s.responseInternalError(err, w)
				return
			}
		}
	}
	w.WriteHeader(http.StatusCreated)
}

func extractToArtifactDir(s *Server, buildId string, part *multipart.Part) error {
	// TODO: find out the right way to unzip multipart.Part in memory
	data, err := ioutil.ReadAll(part)
	if err != nil {
		return err
	}
	zipReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, file := range zipReader.File {
		dest := s.ArtifactFile(buildId, file.FileHeader.Name)
		err := extractArtifactFile(file, dest)
		if err != nil {
			return err
		}
	}
	return nil
}

func extractArtifactFile(file *zip.File, dest string) error {
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
