/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
	return &FileStorage{path: f.Basepath + "/" + schema + "/"}
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
		os.Rename(s.path+"schema.json", s.path+"schema.json.old")
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
	os.MkdirAll(s.path, 0750)
	f, err := os.Create(s.path + shard + "-" + ProcessColumnName(column))
	if err != nil {
		panic(err)
	}
	return f
}

func (s *FileStorage) RemoveColumn(shard string, column string) {
	os.Remove(s.path + shard + "-" + ProcessColumnName(column))
}

func (s *FileStorage) blobPath(hash string) string {
	if len(hash) >= 4 {
		return s.path + "blob/" + hash[:2] + "/" + hash[2:4] + "/" + hash
	}
	return s.path + "blob/" + hash
}

func (s *FileStorage) ReadBlob(hash string) io.ReadCloser {
	f, err := os.Open(s.blobPath(hash))
	if err != nil {
		return ErrorReader{err}
	}
	return f
}

func (s *FileStorage) WriteBlob(hash string) io.WriteCloser {
	p := s.blobPath(hash)
	os.MkdirAll(p[:strings.LastIndex(p, "/")], 0750)
	f, err := os.Create(p)
	if err != nil {
		panic(err)
	}
	return f
}

func (s *FileStorage) DeleteBlob(hash string) {
	os.Remove(s.blobPath(hash))
}

func (s *FileStorage) OpenLog(shard string) PersistenceLogfile {
	os.MkdirAll(s.path, 0750)
	f, err := os.OpenFile(s.path+shard+".log", os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		panic(err)
	}
	return FileLogfile{f}
}

func (s *FileStorage) ReplayLog(shard string) (chan interface{}, PersistenceLogfile) {
	os.MkdirAll(s.path, 0750)
	f, err := os.OpenFile(s.path+shard+".log", os.O_RDWR|os.O_CREATE, 0750)
	if err != nil {
		panic(err)
	}
	replay := make(chan interface{}, 64)
	fi, _ := f.Stat()
	if fi.Size() > 0 {
		go func() {
			defer close(replay)
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				b := scanner.Bytes()
				if string(b) == "" {
					// nop
				} else if string(b[0:7]) == "delete " {
					var idx uint32
					json.Unmarshal(b[7:], &idx)
					replay <- LogEntryDelete{idx}
				} else if string(b[0:7]) == "insert " {
					body := string(b[7:])
					if pos := strings.Index(body, "]["); pos >= 0 {
						// new format: columns ][ values
						var cols []string
						var values [][]scm.Scmer
						json.Unmarshal([]byte(body[:pos+1]), &cols)
						json.Unmarshal([]byte(body[pos+1:]), &values)
						for i := 0; i < len(values); i++ {
							for j := 0; j < len(values[i]); j++ {
								values[i][j] = scm.TransformFromJSON(values[i][j])
							}
						}
						replay <- LogEntryInsert{cols, values}
					} else {
						// fallback/old format: flat array of alternating key/value pairs -> single row
						var flat []interface{}
						if err := json.Unmarshal([]byte(body), &flat); err != nil {
							panic("unknown log sequence: " + string(b))
						}
						if len(flat)%2 != 0 {
							panic("corrupt insert log (odd items): " + string(b))
						}
						cols := make([]string, 0, len(flat)/2)
						row := make([]scm.Scmer, 0, len(flat)/2)
						for i := 0; i < len(flat); i += 2 {
							cols = append(cols, flat[i].(string))
							row = append(row, scm.TransformFromJSON(flat[i+1]))
						}
						replay <- LogEntryInsert{cols, [][]scm.Scmer{row}}
					}
				} else {
					panic("unknown log sequence: " + string(b))
				}
			}
		}()
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
