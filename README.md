# AC: Aho-Corasick String Matching in Go

Package `ahocorasick` provides a high-performance implementation of the Aho-Corasick string matching algorithm in Go. It is designed for efficient multi-pattern searching with specific optimizations for ASCII text.

### Ken Steele Variant (`ACKS`)
*   **Mechanism**: Flattens the state machine into a dense Deterministic Finite Automaton (DFA) table, pre-calculating the next state for every possible input character.
*   **Optimization**: Utilizes **Alphabet Compression** to map only used ASCII characters to dense indices, significantly reducing table size.
*   **Pros**: Extremely fast, deterministic **O(1)** search speed per input byte.
*   **Cons**: Higher memory usage and slower build times due to dense table construction.
*   **Best For**: High-throughput applications where search speed is the top priority.

## Installation

```bash
go get github.com/yanlinLiu0424/ahocorasick
```

## Features

*   **Case-Insensitive Matching**: Supports ASCII case-insensitive matching via the `Caseless` flag.
*   **Single Match Mode**: Option to report a pattern ID only the first time it is found using the `SingleMatch` flag.
*   **Zero-Allocation Scan**: The `Scan` method processes matches via a callback handler, preventing memory allocations associated with result slices.

## Usage

### Basic Search

```go
package main

import (
	"fmt"
	"log"

	"github.com/yanlinLiu0424/ahocorasick"
)

func main() {
	// 1. Initialize the matcher
	matcher := ahocorasick.NewACKS()

	// 2. Add patterns
	// Note: IDs must be non-zero.
	_ = matcher.AddPattern(ahocorasick.Pattern{
		Content: []byte("he"),
		ID:      1,
	})
	_ = matcher.AddPattern(ahocorasick.Pattern{
		Content: []byte("she"),
		ID:      2,
	})
	_ = matcher.AddPattern(ahocorasick.Pattern{
		Content: []byte("HIS"),
		ID:      3,
		Flags:   ahocorasick.Caseless, // Case-insensitive match
	})

	// 3. Build the automaton
	matcher.Build()

	// 4. Search in text
	text := []byte("Ushers his")

	// Option A: Get all match IDs
	matches, err := matcher.Search(text)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Matches:", matches)

	// Option B: Scan with callback (Zero-Allocation)
	err = matcher.Scan(text, func(id uint, from, to uint64) error {
		fmt.Printf("Pattern %d found ending at %d\n", id, to)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

## Performance
Benchmarks are included in the test files. `ACKS` provides high search throughput due to its branch-free state transition logic, making it ideal for read-heavy workloads.

Here are a few benchmark results 

```

BenchmarkACKS_Search_FixedPatterns_50000 	   68469	     16671 ns/op	       0 B/op	       0 allocs/op
BenchmarkACKS_Search_RandomPatterns_10000 	   97682	     13102 ns/op	       0 B/op	       0 allocs/op
```

