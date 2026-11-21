// Copyright (C) 2026 Carl-Philip Hänsch
// SPDX-License-Identifier: GPL-3.0-or-later

package storage

import "io"
import "strings"
import "compress/gzip"
import "github.com/launix-de/memcp/scm"

const (
	blobSampleSlots       = 8
	blobSampleValueLimit  = 32 * 1024
	blobSampleCopyLimit   = 1024 * 1024
	blobExtraInlineBudget = 2 * 1024 * 1024
)

// longTextSample belongs only to the analysis pass. The eight owned strings
// retain at most 256 KiB; reservoir replacement copies at most 1 MiB in total.
// Large source buffers are never retained through borrowed substrings. Values
// above 32 KiB stay blobs, bounding both the probe and additional inline work.
type longTextSample struct {
	values [blobSampleSlots]string
	count  int
	seen   uint64
	bytes  uint64
	random uint64
	copied int
}

func (s *longTextSample) offer(text string) {
	if len(text) <= maxInlineBlobBytes || len(text) > blobSampleValueLimit {
		return
	}
	s.seen++
	s.bytes += uint64(len(text))
	if s.copied+len(text) > blobSampleCopyLimit {
		return
	}
	if s.random == 0 {
		s.random = 0x9e3779b97f4a7c15
	}
	s.random ^= s.random << 13
	s.random ^= s.random >> 7
	s.random ^= s.random << 17
	slot := s.count
	if slot == len(s.values) {
		candidate := s.random % s.seen
		if candidate >= uint64(len(s.values)) {
			return
		}
		slot = int(candidate)
	} else {
		s.count++
	}
	s.values[slot] = strings.Clone(text)
	s.copied += len(text)
}

// inlineLimit estimates the seed once and subsequent pool growth per distinct
// document. Individual gzip objects already deduplicate identical full values;
// a sample with repeated whole documents therefore keeps that existing path.
// Only a clear 20% estimated byte saving enables the larger inline window.
func (s *longTextSample) inlineLimit() int {
	if s == nil || s.count < 4 {
		return maxInlineBlobBytes
	}
	unique := make(map[string]bool, s.count)
	for _, text := range s.values[:s.count] {
		unique[text] = true
	}
	if len(unique) != s.count {
		return maxInlineBlobBytes
	}
	if !sampleSharesChunks(s.values[:s.count]) {
		return maxInlineBlobBytes
	}
	pool := scm.NewSamplingStringConcatBuilder(s.count)
	var writer *gzip.Writer
	compressed := countingWriter{}
	zipBytes, firstBytes, sourceBytes := 0, 0, 0
	worstGrowth, worstLength := uint64(0), uint64(1)
	for i, text := range s.values[:s.count] {
		before := pool.Len()
		pool.Add(text)
		if i == 0 {
			firstBytes = pool.Len()
		} else if growth := uint64(pool.Len() - before); growth*worstLength > worstGrowth*uint64(len(text)) {
			worstGrowth, worstLength = growth, uint64(len(text))
		}
		sourceBytes += len(text)
		compressed.n = 0
		if writer == nil {
			writer = gzip.NewWriter(&compressed)
		} else {
			writer.Reset(&compressed)
		}
		_, _ = io.WriteString(writer, text)
		_ = writer.Close() // countingWriter cannot fail
		// Include the binary hash marker and entry offset, but no filesystem
		// block-rounding advantage: both resident and serialized size matter.
		zipBytes += compressed.n + 34 + 8
	}
	population := min(s.seen, uint64(blobExtraInlineBudget)*s.seen/s.bytes)
	if population < uint64(s.count) {
		return maxInlineBlobBytes
	}
	// A mean incremental cost overfit one heterogeneous PDF shard badly.
	// Extrapolate the worst observed growth per source byte instead, including
	// an entry directory allowance. Strong shared templates still win, while
	// mixed families keep the already competitive per-document gzip path.
	inlineBytes := min(s.bytes, uint64(blobExtraInlineBudget))
	remaining := inlineBytes - uint64(len(s.values[0]))
	poolBytes := uint64(firstBytes) + remaining*worstGrowth/worstLength + population*8
	zipEstimate := uint64(zipBytes) * inlineBytes / uint64(sourceBytes)
	if poolBytes*5 < zipEstimate*4 {
		return blobSampleValueLimit
	}
	return maxInlineBlobBytes
}

// A fixed fingerprint table rejects weakly related samples before allocating a
// trigram index or gzip workspace. Most non-seed documents must share at least
// 75% of their bytes in 16-byte windows. Common PDF headers alone are not enough.
// Hash hits only request the full cost probe; collisions cannot change stored
// bytes. Sliding lookups tolerate shifted text; inserts use every 16th window.
func sampleSharesChunks(texts []string) bool {
	var table [2048]struct {
		hash     uint64
		document uint8
	}
	power := uint64(1)
	for i := 0; i < 15; i++ {
		power *= 257
	}
	sharedDocuments := 0
	required := len(texts) / 2
	for document, text := range texts {
		if len(text) < 16 {
			continue
		}
		hash := uint64(0)
		for i := 0; i < 16; i++ {
			hash = hash*257 + uint64(text[i])
		}
		shared, coveredUntil := 0, 0
		for offset := 0; offset+16 <= len(text); offset++ {
			if offset != 0 {
				hash = (hash-uint64(text[offset-1])*power)*257 + uint64(text[offset+15])
			}
			if document == 0 && offset%16 != 0 {
				continue
			}
			slot := (hash * 0x9e3779b97f4a7c15) >> 53
			for probe := uint64(0); probe < 4; probe++ {
				entry := &table[(slot+probe)&2047]
				if entry.document == 0 {
					if offset%16 == 0 {
						entry.hash, entry.document = hash, uint8(document+1)
					}
					break
				}
				if entry.hash == hash {
					if entry.document != uint8(document+1) {
						shared += offset + 16 - max(offset, coveredUntil)
						coveredUntil = offset + 16
					}
					break
				}
			}
		}
		if shared*4 >= len(text)*3 {
			sharedDocuments++
			if sharedDocuments >= required {
				return true
			}
		}
	}
	return false
}

type countingWriter struct{ n int }

func (w *countingWriter) Write(p []byte) (int, error) {
	w.n += len(p)
	return len(p), nil
}
