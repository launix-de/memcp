/*
Copyright (C) 2026  Carl-Philip Hänsch

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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
)

func s3IntegrationFactory(t *testing.T) *S3Factory {
	t.Helper()
	endpoint := os.Getenv("MEMCP_TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("MEMCP_TEST_S3_ENDPOINT is not set")
	}
	bucket := os.Getenv("MEMCP_TEST_S3_BUCKET")
	if bucket == "" {
		bucket = "memcp-test"
	}
	accessKey := os.Getenv("MEMCP_TEST_S3_ACCESS_KEY")
	secretKey := os.Getenv("MEMCP_TEST_S3_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		t.Fatal("MEMCP_TEST_S3_ACCESS_KEY and MEMCP_TEST_S3_SECRET_KEY are required")
	}
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion("us-east-1"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	)
	if err != nil {
		t.Fatal(err)
	}
	client := s3.NewFromConfig(cfg, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(endpoint)
		options.UsePathStyle = true
	})
	if _, err := client.CreateBucket(context.Background(), &s3.CreateBucketInput{Bucket: aws.String(bucket)}); err != nil {
		var apiErr smithy.APIError
		if !errors.As(err, &apiErr) || (apiErr.ErrorCode() != "BucketAlreadyOwnedByYou" && apiErr.ErrorCode() != "BucketAlreadyExists") {
			t.Fatalf("create S3 test bucket: %v", err)
		}
	}
	return &S3Factory{
		AccessKeyID: accessKey, SecretAccessKey: secretKey,
		Region: "us-east-1", Endpoint: endpoint, Bucket: bucket,
		Prefix: "persistence-contract", ForcePathStyle: true,
	}
}

func testS3Storage(server *httptest.Server) *S3Storage {
	client := s3.New(s3.Options{
		Region:       "us-east-1",
		BaseEndpoint: aws.String(server.URL),
		UsePathStyle: true,
		Credentials:  credentials.NewStaticCredentialsProvider("test", "test", ""),
		Retryer:      aws.NopRetryer{},
	})
	return &S3Storage{
		factory: &S3Factory{Bucket: "bucket"}, prefix: "database",
		client: client, opened: true,
	}
}

func requirePersistenceFailure(t *testing.T, operation func()) {
	t.Helper()
	defer func() {
		if _, ok := recover().(*PersistenceFailure); !ok {
			t.Fatal("operation did not panic with *PersistenceFailure")
		}
	}()
	operation()
}

func TestS3OpenLogDoesNotReplaceUnreadableManifest(t *testing.T) {
	putRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPut {
			putRequests++
		}
		writer.Header().Set("Content-Type", "application/xml")
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte("<Error><Code>SlowDown</Code><Message>retry later</Message></Error>"))
	}))
	defer server.Close()

	requirePersistenceFailure(t, func() { testS3Storage(server).OpenLog("shard") })
	if putRequests != 0 {
		t.Fatalf("unreadable S3 manifest triggered %d replacement PUTs", putRequests)
	}
}

func TestS3OpenLogDoesNotReplaceCorruptManifest(t *testing.T) {
	putRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPut {
			putRequests++
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte("not-json"))
	}))
	defer server.Close()

	requirePersistenceFailure(t, func() { testS3Storage(server).OpenLog("shard") })
	if putRequests != 0 {
		t.Fatalf("corrupt S3 manifest triggered %d replacement PUTs", putRequests)
	}
}

func TestS3OpenLogDoesNotTreatHeadFailureAsEmptySegment(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write([]byte("[0]"))
			return
		}
		writer.Header().Set("Content-Type", "application/xml")
		writer.WriteHeader(http.StatusServiceUnavailable)
		_, _ = writer.Write([]byte("<Error><Code>SlowDown</Code><Message>retry later</Message></Error>"))
	}))
	defer server.Close()

	requirePersistenceFailure(t, func() { testS3Storage(server).OpenLog("shard") })
}

func TestS3LogFlushRetryDoesNotDuplicateAmbiguousPut(t *testing.T) {
	object := []byte("acknowledged-")
	putRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.Method {
		case http.MethodGet:
			writer.WriteHeader(http.StatusOK)
			_, _ = writer.Write(object)
		case http.MethodPut:
			putRequests++
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Errorf("read PUT body: %v", err)
			}
			object = append(object[:0], body...)
			if putRequests == 1 {
				writer.Header().Set("Content-Type", "application/xml")
				writer.WriteHeader(http.StatusServiceUnavailable)
				_, _ = writer.Write([]byte("<Error><Code>SlowDown</Code><Message>response lost</Message></Error>"))
				return
			}
			writer.WriteHeader(http.StatusOK)
		default:
			writer.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	storage := testS3Storage(server)
	logfile := &S3Logfile{
		s: storage, shard: "shard", key: storage.key("shard.log.00000000"),
		offset: uint64(len(object)), flushEveryBytes: 256 * 1024,
	}
	logfile.buf.WriteString("pending")
	if err := logfile.flushLocked(false); err == nil {
		t.Fatal("first PUT should report its ambiguous response failure")
	}
	if err := logfile.flushLocked(false); err != nil {
		t.Fatalf("retry S3 WAL flush: %v", err)
	}
	if got, want := string(object), "acknowledged-pending"; got != want {
		t.Fatalf("S3 WAL after ambiguous retry = %q, want %q", got, want)
	}
}

func readPersistenceObject(t *testing.T, reader io.ReadCloser) []byte {
	t.Helper()
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func writePersistenceObject(t *testing.T, writer io.WriteCloser, data []byte) {
	t.Helper()
	if _, err := writer.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestS3PersistenceStress(t *testing.T) {
	factory := s3IntegrationFactory(t)
	engine := factory.CreateDatabase("stress")
	engine.Remove()
	defer engine.Remove()

	if schema := engine.ReadSchema(); schema != nil {
		t.Fatalf("new S3 schema = %q, want nil", schema)
	}
	engine.WriteSchema([]byte(`{"generation":1}`))
	engine.WriteSchema([]byte(`{"generation":2}`))
	if got := engine.ReadSchema(); !bytes.Equal(got, []byte(`{"generation":2}`)) {
		t.Fatalf("replaced S3 schema = %q", got)
	}

	columnData := bytes.Repeat([]byte("column-data-"), 4096)
	writePersistenceObject(t, engine.WriteColumn("shard", "payload"), columnData)
	if got := readPersistenceObject(t, engine.ReadColumn("shard", "payload")); !bytes.Equal(got, columnData) {
		t.Fatal("S3 column roundtrip changed data")
	}
	engine.RemoveColumn("shard", "payload")
	missingColumn := engine.ReadColumn("shard", "payload")
	if missing, ok := missingColumn.(interface{ Missing() bool }); !ok || !missing.Missing() {
		t.Fatal("removed S3 column is not reported as missing")
	}
	_ = missingColumn.Close()

	// Exceed the default 1000-key ListObjectsV2 page to exercise pagination.
	const blobs = 1005
	const workers = 16
	var wait sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		wait.Add(1)
		go func(worker int) {
			defer wait.Done()
			for index := worker; index < blobs; index += workers {
				name := fmt.Sprintf("%04d", index)
				writer := engine.WriteBlob(name)
				if _, err := writer.Write([]byte("blob-" + name)); err != nil {
					panic(err)
				}
				if err := writer.Close(); err != nil {
					panic(err)
				}
			}
		}(worker)
	}
	wait.Wait()
	seen := make(map[string]struct{}, blobs)
	engine.WalkBlobs(func(hash string) { seen[hash] = struct{}{} })
	if len(seen) != blobs {
		t.Fatalf("WalkBlobs returned %d objects, want %d", len(seen), blobs)
	}
	if got := readPersistenceObject(t, engine.ReadBlob("0500")); string(got) != "blob-0500" {
		t.Fatalf("S3 blob roundtrip = %q", got)
	}

	logfile := engine.OpenLog("shard")
	for index := 0; index < 250; index++ {
		logfile.Write(LogEntryDelete{idx: uint32(index), txID: "tx-a"})
	}
	logfile.Write(LogEntryCommit{txID: "tx-a"})
	logfile.Flush(true)
	logfile.Close()

	reopened := factory.CreateDatabase("stress")
	committed, entries, appendLog := reopened.ReplayLog("shard")
	if _, ok := committed["tx-a"]; !ok {
		t.Fatal("replayed S3 WAL lost commit marker")
	}
	entryCount := 0
	for range entries {
		entryCount++
	}
	appendLog.Close()
	if entryCount != 251 {
		t.Fatalf("replayed S3 WAL entries = %d, want 251", entryCount)
	}

	replacement := reopened.SwapLog("shard", []interface{}{
		LogEntryDelete{idx: 999, txID: "tx-b"}, LogEntryCommit{txID: "tx-b"},
	}, true)
	replacement.Close()
	committed, entries, appendLog = reopened.ReplayLog("shard")
	if _, ok := committed["tx-b"]; !ok || len(committed) != 1 {
		t.Fatalf("commits after S3 WAL swap = %v", committed)
	}
	entryCount = 0
	for range entries {
		entryCount++
	}
	appendLog.Close()
	if entryCount != 2 {
		t.Fatalf("entries after S3 WAL swap = %d, want 2", entryCount)
	}

	reopened.RemoveLog("shard")
	committed, entries, appendLog = reopened.ReplayLog("shard")
	for range entries {
		t.Fatal("removed S3 WAL replayed an entry")
	}
	appendLog.Close()
	if len(committed) != 0 {
		t.Fatalf("removed S3 WAL retained commits: %v", committed)
	}
}
