package ahocorasick

import (
	"bytes"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func TestACKS_Search_Basic(t *testing.T) {
	ac := NewACKS()
	ac.AddPattern(mkPat("he", 1, 0))
	ac.AddPattern(mkPat("she", 2, 0))
	ac.AddPattern(mkPat("his", 3, 0))
	ac.Build()

	text := []byte("ushers")
	matches, err := ac.Search(text)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	expected := []uint{1, 2}
	sortSlice(matches)
	sortSlice(expected)

	if !reflect.DeepEqual(matches, expected) {
		t.Errorf("Expected %v, got %v", expected, matches)
	}
}

func TestACKS_Search_Caseless(t *testing.T) {
	ac := NewACKS()
	ac.AddPattern(mkPat("AbC", 10, Caseless))
	ac.Build()

	text := []byte("abC")
	matches, err := ac.Search(text)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	expected := []uint{10}
	if !reflect.DeepEqual(matches, expected) {
		t.Errorf("Expected %v, got %v", expected, matches)
	}
}

func TestACKS_Search_SingleMatch(t *testing.T) {
	ac := NewACKS()
	ac.AddPattern(mkPat("foo", 100, SingleMatch))
	ac.Build()

	text := []byte("foofoo")
	matches, err := ac.Search(text)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("Expected 1 match, got %d", len(matches))
	}
	if len(matches) > 0 && matches[0] != 100 {
		t.Errorf("Expected match ID 100, got %d", matches[0])
	}
}

func TestACKS_Search_Mixed(t *testing.T) {
	ac := NewACKS()
	ac.AddPattern(mkPat("foo", 1, SingleMatch))
	ac.AddPattern(mkPat("bar", 2, Caseless))
	ac.Build()

	text := []byte("fooBarFoo")
	matches, err := ac.Search(text)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	expected := []uint{1, 2}
	sortSlice(matches)
	sortSlice(expected)

	if !reflect.DeepEqual(matches, expected) {
		t.Errorf("Expected %v, got %v", expected, matches)
	}
}

func TestACKS_Search_CaseSensitive(t *testing.T) {
	ac := NewACKS()
	ac.AddPattern(mkPat("abc", 1, 0))
	ac.Build()

	text := []byte("ABC")
	matches, err := ac.Search(text)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(matches) != 0 {
		t.Errorf("Expected no matches, got %v", matches)
	}
}

func mkPat(content string, id uint, flags Flag) Pattern {
	return Pattern{
		Content: []byte(content),
		ID:      id,
		Flags:   flags,
		strlen:  len(content),
	}
}

func sortSlice(s []uint) {
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
}

func BenchmarkACKS_Search_FixedPatterns(b *testing.B) {
	ac := NewACKS()
	numPatterns := 50000
	for i := 0; i < numPatterns; i++ {
		s := fmt.Sprintf("FixedString%d", i)
		_ = ac.AddPattern(mkPat(s, uint(i+1), Caseless))
	}
	ac.Build()

	var buffer bytes.Buffer
	for i := 0; i < 200; i++ {
		buffer.WriteString(fmt.Sprintf("noise_FixedString%d_data ", i%numPatterns))
	}
	text := buffer.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	handler := MatchedHandler(func(id uint, from, to uint64) error { return nil })
	for i := 0; i < b.N; i++ {
		_ = ac.Scan(text, handler)
	}
}

func BenchmarkACKS_Search_RandomPatterns(b *testing.B) {
	ac := NewACKS()
	numPatterns := 10000
	patterns := make([]string, 0, numPatterns)
	for i := 0; i < numPatterns; i++ {
		s := randomString(10)
		patterns = append(patterns, s)
		_ = ac.AddPattern(mkPat(s, uint(i+1), 0))
	}
	ac.Build()

	var buffer bytes.Buffer
	for i := 0; i < 100; i++ {
		buffer.WriteString(randomString(10))
		buffer.WriteString(patterns[rand.Intn(numPatterns)])
	}
	text := buffer.Bytes()

	b.ReportAllocs()
	b.ResetTimer()
	handler := MatchedHandler(func(id uint, from, to uint64) error { return nil })
	for i := 0; i < b.N; i++ {
		_ = ac.Scan(text, handler)
	}
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomString(n int) string {
	sb := make([]byte, n)
	for i := range sb {
		sb[i] = charset[rand.Intn(len(charset))]
	}
	return string(sb)
}
