/*
Copyright (C) 2024  Carl-Philip Hänsch

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package storage

import "io"
import "os"
import "fmt"
import "bufio"
import "bytes"
import "strings"
import "crypto/sha256"
import "encoding/json"
import "github.com/launix-de/memcp/scm"

type FileStorage struct {
	path string
}

type FileFactory struct {
	Basepath string
}

// helper for long column names
func ProcessColumnName(col string) string {
	if len(col) < 64 {
		return col
	} else {
		hashsum := sha256.Sum256([]byte(col))
		return fmt.Sprintf("%x", hashsum[:8])
	}
}

func (f *FileFactory) CreateDatabase(schema string) PersistenceEngine {
	return &FileStorage{f.Basepath + "/" + schema + "/"}
}

func (f *FileStorage) ReadSchema() []byte {
	jsonbytes, _ := os.ReadFile(f.path + "schema.json")
	if len(jsonbytes) == 0 {
		// try to load backup (in case of failure while save)
		jsonbytes, _ = os.ReadFile(f.path + "schema.json.old")
	}
	return jsonbytes
}

func (s *FileStorage) WriteSchema(jsonbytes []byte) {
	os.MkdirAll(s.path, 0750)
	if stat, err := os.Stat(s.path + "schema.json"); err == nil && stat.Size() > 0 {
		// rescue a copy of schema.json in case the schema is not serializable
		os.Rename(s.path + "schema.json", s.path + "schema.json.old")
	}
	f, err := os.Create(s.path + "schema.json")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.Write(jsonbytes)
}

func (s *FileStorage) ReadColumn(shard string, column string) io.ReadCloser {
	//f, err := os.C
	f, err := os.Open(s.path + shard + "-" + ProcessColumnName(column))
	if err != nil {
		// file does not exist -> no data available
		return ErrorReader{err}
	}
	return f
}

func (s *FileStorage) WriteColumn(shard string, column string) io.WriteCloser {
	f, err := os.Create(s.path + shard + "-" + ProcessColumnName(column))
	if err != nil {
		panic(err)
	}
	return f
}

func (s *FileStorage) RemoveColumn(shard string, column string) {
	os.Remove(s.path + shard + "-" + ProcessColumnName(column))
}

func (s *FileStorage) OpenLog(shard string) PersistenceLogfile {
	f, err := os.OpenFile(s.path + shard + ".log", os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		panic(err)
	}
	return FileLogfile{f}
}

func (s *FileStorage) ReplayLog(shard string) (chan interface{}, PersistenceLogfile) {
	f, err := os.OpenFile(s.path + shard + ".log", os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		panic(err)
	}
	replay := make(chan interface{}, 8)
	fi, _ := f.Stat()
	if fi.Size() > 0 {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			b := scanner.Bytes()
			if string(b) == "" {
				// nop
			} else if string(b[0:7]) == "delete " {
				var idx uint
				json.Unmarshal(b[7:], &idx)
				replay <- LogEntryDelete{idx}
			} else if string(b[0:7]) == "insert " {
				split := strings.Index(string(b), "][") + 1
				var cols []string
				var values [][]scm.Scmer
				json.Unmarshal(b[7:split], &cols)
				json.Unmarshal(b[split:], &values)
				replay <- LogEntryInsert{cols, values}
			} else {
				panic("unknown log sequence: " + string(b))
			}
		}
	} else {
		close(replay)
	}
	return replay, FileLogfile{f}
}

func (s *FileStorage) RemoveLog(shard string) {
	os.Remove(s.path + shard + ".log")
}

type FileLogfile struct {
	w *os.File
}
func (w FileLogfile) Write(logentry interface{}) {
	switch l := logentry.(type) {
		case LogEntryDelete:
			var b bytes.Buffer
			b.WriteString("delete ")
			tmp, _ := json.Marshal(l.idx)
			b.Write(tmp)
			b.WriteString("\n")
			w.w.Write(b.Bytes())
		case LogEntryInsert:
			var b bytes.Buffer
			b.WriteString("insert ")
			tmp, _ := json.Marshal(l.cols)
			b.Write(tmp)
			tmp, _ = json.Marshal(l.values)
			b.Write(tmp)
			b.WriteString("\n")
			w.w.Write(b.Bytes())
	}
}
func (w FileLogfile) Sync() {
	w.w.Sync()
}
func (w FileLogfile) Close() {
	w.w.Close()
}

func (s *FileStorage) Remove() {
	os.RemoveAll(s.path)
}
