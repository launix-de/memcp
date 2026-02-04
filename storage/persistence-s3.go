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

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/launix-de/memcp/scm"
)

// S3 layout:
//  - schema:   <prefix>/schema.json
//  - column:   <prefix>/<shard>-<colhash>
//  - logs:     <prefix>/<shard>.log.<seg8>
//             segments are append-only, we always write to the last segment.
//
// S3 does not support append; we buffer and replace objects on sync.

type S3Factory struct {
	AccessKeyID     string // AWS or S3-compatible access key
	SecretAccessKey string // AWS or S3-compatible secret key
	Region          string // AWS region (e.g., "us-east-1")
	Endpoint        string // Custom endpoint for S3-compatible storage (MinIO, etc.)
	Bucket          string // S3 bucket name
	Prefix          string // Object key prefix
	ForcePathStyle  bool   // Use path-style URLs (required for MinIO)
}

func (f *S3Factory) CreateDatabase(schema string) PersistenceEngine {
	pfx := strings.TrimSuffix(f.Prefix, "/")
	if pfx != "" {
		pfx = pfx + "/" + schema
	} else {
		pfx = schema
	}
	return NewS3Storage(f, pfx)
}

type S3Storage struct {
	factory *S3Factory
	prefix  string

	mu     sync.Mutex
	client *s3.Client
	opened bool
}

func NewS3Storage(f *S3Factory, prefix string) *S3Storage {
	return &S3Storage{factory: f, prefix: prefix}
}

func (s *S3Storage) ensureOpen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.opened {
		return
	}

	ctx := context.Background()

	// Build AWS config with custom credentials
	var opts []func(*config.LoadOptions) error

	if s.factory.Region != "" {
		opts = append(opts, config.WithRegion(s.factory.Region))
	}

	if s.factory.AccessKeyID != "" && s.factory.SecretAccessKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				s.factory.AccessKeyID,
				s.factory.SecretAccessKey,
				"", // session token
			),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		panic(fmt.Sprintf("S3Storage: failed to load AWS config: %v", err))
	}

	// Build S3 client options
	var s3Opts []func(*s3.Options)

	if s.factory.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s.factory.Endpoint)
		})
	}

	if s.factory.ForcePathStyle {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	s.client = s3.NewFromConfig(cfg, s3Opts...)
	s.opened = true
}

func (s *S3Storage) key(name string) string {
	return s.prefix + "/" + name
}

func (s *S3Storage) ReadSchema() []byte {
	s.ensureOpen()
	key := s.key("schema.json")

	resp, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.factory.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	return data
}

func (s *S3Storage) WriteSchema(schema []byte) {
	s.ensureOpen()
	key := s.key("schema.json")

	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.factory.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(schema),
	})
	if err != nil {
		panic(fmt.Sprintf("S3Storage: failed to write schema: %v", err))
	}
}

func (s *S3Storage) ReadColumn(shard string, column string) io.ReadCloser {
	s.ensureOpen()
	key := s.key(shard + "-" + ProcessColumnName(column))

	resp, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.factory.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return ErrorReader{err}
	}
	return resp.Body
}

type s3WriteCloser struct {
	s      *S3Storage
	key    string
	buf    bytes.Buffer
	closed bool
}

func (w *s3WriteCloser) Write(p []byte) (int, error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}
	return w.buf.Write(p)
}

func (w *s3WriteCloser) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	_, err := w.s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(w.s.factory.Bucket),
		Key:    aws.String(w.key),
		Body:   bytes.NewReader(w.buf.Bytes()),
	})
	return err
}

func (s *S3Storage) WriteColumn(shard string, column string) io.WriteCloser {
	s.ensureOpen()
	key := s.key(shard + "-" + ProcessColumnName(column))
	return &s3WriteCloser{s: s, key: key}
}

func (s *S3Storage) RemoveColumn(shard string, column string) {
	s.ensureOpen()
	key := s.key(shard + "-" + ProcessColumnName(column))
	_, _ = s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.factory.Bucket),
		Key:    aws.String(key),
	})
}

func (s *S3Storage) Remove() {
	// List and delete all objects with prefix
	s.ensureOpen()

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.factory.Bucket),
		Prefix: aws.String(s.prefix + "/"),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(context.Background())
		if err != nil {
			break
		}
		for _, obj := range page.Contents {
			_, _ = s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
				Bucket: aws.String(s.factory.Bucket),
				Key:    obj.Key,
			})
		}
	}
}

// -------------------- Logs --------------------

type S3Logfile struct {
	s     *S3Storage
	shard string

	mu     sync.Mutex
	seg    uint32
	key    string
	buf    bytes.Buffer
	offset uint64 // logical offset within segment (for segment rollover logic)

	flushEveryBytes int
}

func (s *S3Storage) OpenLog(shard string) PersistenceLogfile {
	s.ensureOpen()
	lf, err := openOrCreateS3Logfile(s, shard)
	if err != nil {
		panic(err)
	}
	return lf
}

func (s *S3Storage) ReplayLog(shard string) (chan interface{}, PersistenceLogfile) {
	s.ensureOpen()

	out := make(chan interface{}, 64)

	go func() {
		defer close(out)
		segments, err := listS3LogSegments(s, shard)
		if err != nil {
			return
		}
		sort.Slice(segments, func(i, j int) bool { return segments[i].seg < segments[j].seg })

		for _, seg := range segments {
			resp, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
				Bucket: aws.String(s.factory.Bucket),
				Key:    aws.String(seg.key),
			})
			if err != nil {
				continue
			}
			data, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil || len(data) == 0 {
				continue
			}
			decodeS3LogStream(data, out)
		}
	}()

	lf, err := openOrCreateS3Logfile(s, shard)
	if err != nil {
		panic(err)
	}
	return out, lf
}

func (s *S3Storage) RemoveLog(shard string) {
	s.ensureOpen()
	segments, _ := listS3LogSegments(s, shard)
	for _, seg := range segments {
		_, _ = s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
			Bucket: aws.String(s.factory.Bucket),
			Key:    aws.String(seg.key),
		})
	}
	// Also remove manifest
	manifestKey := s.key(fmt.Sprintf("%s.log.manifest", shard))
	_, _ = s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.factory.Bucket),
		Key:    aws.String(manifestKey),
	})
}

type s3LogSegInfo struct {
	seg uint32
	key string
}

func listS3LogSegments(s *S3Storage, shard string) ([]s3LogSegInfo, error) {
	manifestKey := s.key(fmt.Sprintf("%s.log.manifest", shard))

	resp, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.factory.Bucket),
		Key:    aws.String(manifestKey),
	})
	if err != nil {
		return nil, fmt.Errorf("no manifest")
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil || len(raw) == 0 {
		return nil, fmt.Errorf("empty manifest")
	}

	var segs []uint32
	if err := json.Unmarshal(raw, &segs); err != nil {
		return nil, err
	}

	out := make([]s3LogSegInfo, 0, len(segs))
	for _, seg := range segs {
		out = append(out, s3LogSegInfo{
			seg: seg,
			key: s.key(fmt.Sprintf("%s.log.%08d", shard, seg)),
		})
	}
	return out, nil
}

func writeS3LogManifest(s *S3Storage, shard string, segs []uint32) error {
	manifestKey := s.key(fmt.Sprintf("%s.log.manifest", shard))
	raw, _ := json.Marshal(segs)
	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.factory.Bucket),
		Key:    aws.String(manifestKey),
		Body:   bytes.NewReader(raw),
	})
	return err
}

func openOrCreateS3Logfile(s *S3Storage, shard string) (*S3Logfile, error) {
	segs, _ := listS3LogSegments(s, shard)
	var seg uint32
	var all []uint32
	if len(segs) == 0 {
		seg = 0
		all = []uint32{0}
		if err := writeS3LogManifest(s, shard, all); err != nil {
			return nil, err
		}
	} else {
		sort.Slice(segs, func(i, j int) bool { return segs[i].seg < segs[j].seg })
		seg = segs[len(segs)-1].seg
		all = make([]uint32, 0, len(segs))
		for _, si := range segs {
			all = append(all, si.seg)
		}
	}

	key := s.key(fmt.Sprintf("%s.log.%08d", shard, seg))

	// Get current size
	var offset uint64
	head, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(s.factory.Bucket),
		Key:    aws.String(key),
	})
	if err == nil && head.ContentLength != nil {
		offset = uint64(*head.ContentLength)
	}

	return &S3Logfile{
		s:               s,
		shard:           shard,
		seg:             seg,
		key:             key,
		offset:          offset,
		flushEveryBytes: 256 * 1024,
	}, nil
}

// Log encoding (same as Ceph)
type s3EncDelete struct {
	T   string `json:"t"`
	Idx uint   `json:"idx"`
}
type s3EncInsert struct {
	T      string        `json:"t"`
	Cols   []string      `json:"cols"`
	Values [][]scm.Scmer `json:"values"`
}

func encodeS3LogEntry(e interface{}) []byte {
	var payload []byte
	switch l := e.(type) {
	case LogEntryDelete:
		tmp, _ := json.Marshal(s3EncDelete{T: "delete", Idx: l.idx})
		payload = tmp
	case LogEntryInsert:
		tmp, _ := json.Marshal(s3EncInsert{T: "insert", Cols: l.cols, Values: l.values})
		payload = tmp
	default:
		return nil
	}

	var frame bytes.Buffer
	_ = binary.Write(&frame, binary.LittleEndian, uint32(len(payload)))
	frame.Write(payload)
	return frame.Bytes()
}

func decodeS3LogStream(data []byte, out chan interface{}) {
	i := 0
	for {
		if i+4 > len(data) {
			return
		}
		n := int(binary.LittleEndian.Uint32(data[i : i+4]))
		i += 4
		if n <= 0 || i+n > len(data) {
			return
		}
		payload := data[i : i+n]
		i += n

		var head struct {
			T string `json:"t"`
		}
		if err := json.Unmarshal(payload, &head); err != nil {
			continue
		}
		switch head.T {
		case "delete":
			var d s3EncDelete
			if json.Unmarshal(payload, &d) == nil {
				out <- LogEntryDelete{idx: d.Idx}
			}
		case "insert":
			var ins s3EncInsert
			if json.Unmarshal(payload, &ins) == nil {
				for r := 0; r < len(ins.Values); r++ {
					for c := 0; c < len(ins.Values[r]); c++ {
						ins.Values[r][c] = scm.TransformFromJSON(ins.Values[r][c])
					}
				}
				out <- LogEntryInsert{cols: ins.Cols, values: ins.Values}
			}
		}
	}
}

func (w *S3Logfile) Write(logentry interface{}) {
	frame := encodeS3LogEntry(logentry)
	if len(frame) == 0 {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.buf.Write(frame)

	if w.buf.Len() >= w.flushEveryBytes {
		_ = w.flushLocked(false)
	}
}

func (w *S3Logfile) Sync() {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.flushLocked(true)
}

func (w *S3Logfile) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	_ = w.flushLocked(true)
}

func (w *S3Logfile) flushLocked(force bool) error {
	if w.buf.Len() == 0 {
		return nil
	}

	const maxSeg = 64 * 1024 * 1024
	if w.offset+uint64(w.buf.Len()) > maxSeg {
		// Roll over to next segment
		next := w.seg + 1
		nextKey := w.s.key(fmt.Sprintf("%s.log.%08d", w.shard, next))

		segs, _ := listS3LogSegments(w.s, w.shard)
		var all []uint32
		for _, si := range segs {
			all = append(all, si.seg)
		}
		all = append(all, next)
		_ = writeS3LogManifest(w.s, w.shard, all)

		w.seg = next
		w.key = nextKey
		w.offset = 0
	}

	// S3 doesn't support append, so we need to read-modify-write for logs
	// For efficiency, we keep a local buffer and write the full segment
	var existing []byte
	if w.offset > 0 {
		resp, err := w.s.client.GetObject(context.Background(), &s3.GetObjectInput{
			Bucket: aws.String(w.s.factory.Bucket),
			Key:    aws.String(w.key),
		})
		if err == nil {
			existing, _ = io.ReadAll(resp.Body)
			resp.Body.Close()
		}
	}

	newData := append(existing, w.buf.Bytes()...)
	_, err := w.s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(w.s.factory.Bucket),
		Key:    aws.String(w.key),
		Body:   bytes.NewReader(newData),
	})
	if err != nil {
		return err
	}

	w.offset += uint64(w.buf.Len())
	w.buf.Reset()

	if force {
		time.Sleep(0) // placeholder
	}
	return nil
}
