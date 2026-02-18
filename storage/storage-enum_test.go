package storage

import (
	"math/rand"
	"testing"

	"github.com/launix-de/memcp/scm"
)

// buildEnum constructs a StorageEnum from a value generator function.
func buildEnum(n int, gen func(i int) scm.Scmer) *StorageEnum {
	s := new(StorageEnum)
	s.prepare()
	for i := 0; i < n; i++ {
		s.scan(uint32(i), gen(i))
	}
	s.init(uint32(n))
	for i := 0; i < n; i++ {
		s.build(uint32(i), gen(i))
	}
	s.finish()
	return s
}

// --- Correctness tests ---

func TestEnumGetValueReadOnly(t *testing.T) {
	// Verify GetValue is purely read-only (no mutable state on struct)
	n := 1000
	gen := func(i int) scm.Scmer {
		if i%100 == 99 {
			return scm.NewInt(1)
		}
		return scm.NewInt(0)
	}
	s := buildEnum(n, gen)

	// Random access pattern — should work without any cache
	indices := []int{999, 0, 500, 99, 100, 999, 1, 500}
	for _, idx := range indices {
		got := s.GetValue(uint32(idx))
		want := gen(idx)
		if !scm.Equal(got, want) {
			t.Fatalf("GetValue(%d) = %v, want %v", idx, got, want)
		}
	}
}

func TestEnumSequentialCorrectness(t *testing.T) {
	// Boolean skewed 99/1
	n := 10000
	gen := func(i int) scm.Scmer {
		if i%100 == 99 {
			return scm.NewInt(1)
		}
		return scm.NewInt(0)
	}
	s := buildEnum(n, gen)

	for i := 0; i < n; i++ {
		got := s.GetValue(uint32(i))
		want := gen(i)
		if !scm.Equal(got, want) {
			t.Fatalf("GetValue(%d) = %v, want %v", i, got, want)
		}
	}
}

func TestEnumRandomCorrectness(t *testing.T) {
	n := 10000
	gen := func(i int) scm.Scmer {
		if i%100 == 99 {
			return scm.NewInt(1)
		}
		return scm.NewInt(0)
	}
	s := buildEnum(n, gen)

	rng := rand.New(rand.NewSource(42))
	for trial := 0; trial < n; trial++ {
		idx := rng.Intn(n)
		got := s.GetValue(uint32(idx))
		want := gen(idx)
		if !scm.Equal(got, want) {
			t.Fatalf("GetValue(%d) = %v, want %v", idx, got, want)
		}
	}
}

func TestEnumCachedCorrectness(t *testing.T) {
	n := 10000
	gen := func(i int) scm.Scmer {
		if i%100 == 99 {
			return scm.NewInt(1)
		}
		return scm.NewInt(0)
	}
	s := buildEnum(n, gen)

	var cache EnumDecodeCache
	for i := 0; i < n; i++ {
		got := s.GetValueCached(uint32(i), &cache)
		want := gen(i)
		if !scm.Equal(got, want) {
			t.Fatalf("GetValueCached(%d) = %v, want %v", i, got, want)
		}
	}
}

func TestEnumGetCachedReader(t *testing.T) {
	n := 10000
	gen := func(i int) scm.Scmer {
		if i%100 == 99 {
			return scm.NewInt(1)
		}
		return scm.NewInt(0)
	}
	s := buildEnum(n, gen)

	// GetCachedReader should return a cachedEnumReader, not self
	reader := s.GetCachedReader()
	if reader == ColumnReader(s) {
		t.Fatal("GetCachedReader() returned self, expected cachedEnumReader wrapper")
	}
	for i := 0; i < n; i++ {
		got := reader.GetValue(uint32(i))
		want := gen(i)
		if !scm.Equal(got, want) {
			t.Fatalf("cachedReader.GetValue(%d) = %v, want %v", i, got, want)
		}
	}
}

// --- proposeCompression entropy filter tests ---

func TestEnumProposalSkewed(t *testing.T) {
	// 99/1 boolean should propose StorageEnum
	s := new(StorageSCMER)
	s.prepare()
	for i := 0; i < 1000; i++ {
		if i%100 == 99 {
			s.scan(uint32(i), scm.NewInt(1))
		} else {
			s.scan(uint32(i), scm.NewInt(0))
		}
	}
	result := s.proposeCompression(1000)
	if _, ok := result.(*StorageEnum); !ok {
		t.Errorf("99/1 skewed bool: expected *StorageEnum, got %T", result)
	}
}

func TestEnumProposalBalancedBool(t *testing.T) {
	// 50/50 boolean should NOT propose StorageEnum (balanced → PFOR is better)
	s := new(StorageSCMER)
	s.prepare()
	for i := 0; i < 1000; i++ {
		s.scan(uint32(i), scm.NewInt(int64(i%2)))
	}
	result := s.proposeCompression(1000)
	if _, ok := result.(*StorageEnum); ok {
		t.Errorf("50/50 balanced bool: should NOT propose StorageEnum, got %T", result)
	}
}

func TestEnumProposalUniformFour(t *testing.T) {
	// 4-way uniform should NOT propose StorageEnum
	s := new(StorageSCMER)
	s.prepare()
	for i := 0; i < 1000; i++ {
		s.scan(uint32(i), scm.NewInt(int64(i%4)))
	}
	result := s.proposeCompression(1000)
	if _, ok := result.(*StorageEnum); ok {
		t.Errorf("4-way uniform: should NOT propose StorageEnum, got %T", result)
	}
}

func TestEnumProposalUniformStrings(t *testing.T) {
	// 4-way uniform strings should NOT propose StorageEnum, should go to StorageString (→ dict+PFOR)
	s := new(StorageSCMER)
	s.prepare()
	vals := []string{"alpha", "beta", "gamma", "delta"}
	for i := 0; i < 1000; i++ {
		s.scan(uint32(i), scm.NewString(vals[i%4]))
	}
	result := s.proposeCompression(1000)
	if _, ok := result.(*StorageEnum); ok {
		t.Errorf("4-way uniform strings: should NOT propose StorageEnum, got %T", result)
	}
	// should fall through to StorageString
	if _, ok := result.(*StorageString); !ok {
		t.Errorf("4-way uniform strings: expected *StorageString, got %T", result)
	}
}

func TestEnumProposalEightSkewed(t *testing.T) {
	// 8-way skewed (50/20/10/7/5/4/3/1) should propose StorageEnum
	s := new(StorageSCMER)
	s.prepare()
	for i := 0; i < 1000; i++ {
		s.scan(uint32(i), genEightSkewed(i))
	}
	result := s.proposeCompression(1000)
	if _, ok := result.(*StorageEnum); !ok {
		t.Errorf("8-way skewed: expected *StorageEnum, got %T", result)
	}
}

// --- Benchmark helpers ---

// generators for different distributions
func genBoolSkewed(i int) scm.Scmer {
	// 99% zero, 1% one
	if i%100 == 99 {
		return scm.NewInt(1)
	}
	return scm.NewInt(0)
}

func genBoolBalanced(i int) scm.Scmer {
	// 50/50
	if i%2 == 0 {
		return scm.NewInt(0)
	}
	return scm.NewInt(1)
}

func genFourUniform(i int) scm.Scmer {
	return scm.NewInt(int64(i % 4))
}

func genEightSkewed(i int) scm.Scmer {
	// skewed toward 0: 50% 0, 20% 1, 10% 2, 7% 3, 5% 4, 4% 5, 3% 6, 1% 7
	v := i % 100
	switch {
	case v < 50:
		return scm.NewInt(0)
	case v < 70:
		return scm.NewInt(1)
	case v < 80:
		return scm.NewInt(2)
	case v < 87:
		return scm.NewInt(3)
	case v < 92:
		return scm.NewInt(4)
	case v < 96:
		return scm.NewInt(5)
	case v < 99:
		return scm.NewInt(6)
	default:
		return scm.NewInt(7)
	}
}

type benchConfig struct {
	name string
	n    int
	gen  func(int) scm.Scmer
}

var configs = []benchConfig{
	{"Bool99_1_10K", 10000, genBoolSkewed},
	{"Bool99_1_100K", 100000, genBoolSkewed},
	{"Bool50_50_10K", 10000, genBoolBalanced},
	{"Bool50_50_100K", 100000, genBoolBalanced},
	{"Four_uniform_10K", 10000, genFourUniform},
	{"Four_uniform_100K", 100000, genFourUniform},
	{"Eight_skewed_10K", 10000, genEightSkewed},
	{"Eight_skewed_100K", 100000, genEightSkewed},
}

// --- Sequential scan benchmarks ---

func BenchmarkEnumSequential(b *testing.B) {
	for _, cfg := range configs {
		s := buildEnum(cfg.n, cfg.gen)
		b.Run(cfg.name, func(b *testing.B) {
			b.ResetTimer()
			for iter := 0; iter < b.N; iter++ {
				for i := 0; i < cfg.n; i++ {
					s.GetValue(uint32(i))
				}
			}
		})
	}
}

func BenchmarkEnumSequentialCached(b *testing.B) {
	for _, cfg := range configs {
		s := buildEnum(cfg.n, cfg.gen)
		b.Run(cfg.name, func(b *testing.B) {
			b.ResetTimer()
			for iter := 0; iter < b.N; iter++ {
				var cache EnumDecodeCache
				for i := 0; i < cfg.n; i++ {
					s.GetValueCached(uint32(i), &cache)
				}
			}
		})
	}
}

// --- Random access benchmarks ---

func BenchmarkEnumRandom(b *testing.B) {
	for _, cfg := range configs {
		s := buildEnum(cfg.n, cfg.gen)
		rng := rand.New(rand.NewSource(42))
		indices := make([]uint32, cfg.n)
		for i := range indices {
			indices[i] = uint32(rng.Intn(cfg.n))
		}
		b.Run(cfg.name, func(b *testing.B) {
			b.ResetTimer()
			for iter := 0; iter < b.N; iter++ {
				for _, idx := range indices {
					s.GetValue(uint32(idx))
				}
			}
		})
	}
}

func BenchmarkEnumRandomCached(b *testing.B) {
	for _, cfg := range configs {
		s := buildEnum(cfg.n, cfg.gen)
		rng := rand.New(rand.NewSource(42))
		indices := make([]uint32, cfg.n)
		for i := range indices {
			indices[i] = uint32(rng.Intn(cfg.n))
		}
		b.Run(cfg.name, func(b *testing.B) {
			b.ResetTimer()
			for iter := 0; iter < b.N; iter++ {
				var cache EnumDecodeCache
				for _, idx := range indices {
					s.GetValueCached(uint32(idx), &cache)
				}
			}
		})
	}
}

// --- Per-element benchmarks for ns/op comparison ---

func BenchmarkEnumPerElem(b *testing.B) {
	type variant struct {
		name string
		fn   func(s *StorageEnum, n int, b *testing.B)
	}
	variants := []variant{
		{"Sequential", func(s *StorageEnum, n int, b *testing.B) {
			for i := 0; i < b.N; i++ {
				s.GetValue(uint32(i % n))
			}
		}},
		{"SequentialCached", func(s *StorageEnum, n int, b *testing.B) {
			var cache EnumDecodeCache
			for i := 0; i < b.N; i++ {
				s.GetValueCached(uint32(i%n), &cache)
			}
		}},
		{"Random", func(s *StorageEnum, n int, b *testing.B) {
			rng := rand.New(rand.NewSource(99))
			for i := 0; i < b.N; i++ {
				s.GetValue(uint32(rng.Intn(n)))
			}
		}},
	}

	for _, cfg := range configs {
		s := buildEnum(cfg.n, cfg.gen)
		for _, v := range variants {
			b.Run(cfg.name+"/"+v.name, func(b *testing.B) {
				v.fn(s, cfg.n, b)
			})
		}
	}
}
