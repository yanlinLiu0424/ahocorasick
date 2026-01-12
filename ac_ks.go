package ahocorasick

import (
	"bytes"
)

type MatchedHandler func(id uint, from, to uint64) error
type matchedPattern func(pos uint64, ps Pattern) error

type Flag uint

const (
	Caseless Flag = 1 << iota // Caseless represents set case-insensitive matching.
	SingleMatch
)

type Pattern struct {
	Content []byte
	ID      uint // ID
	Flags   Flag // Caseless represents set case-insensitive matching.
	strlen  int
}

// ACKS represents the Aho-Corasick Ken Steele matcher
type ACKS struct {
	patterns       []Pattern
	translateTable [256]uint8
	alphabetSize   int

	// stateTable is a flattened 2D array: stateTable[state * alphabetSize + char]
	stateTable []int32

	// outputTable stores pattern IDs for each state.
	// Using a slice of slices for O(1) access by state index.
	outputTable    [][]int
	size           int
	maxID          uint
	stateCount     int
	hasSingleMatch bool
}

func NewACKS() *ACKS {
	return &ACKS{
		outputTable: make([][]int, 0),
	}
}

func (ac *ACKS) AddPattern(p Pattern) error {
	p.strlen = len(p.Content)
	ac.patterns = append(ac.patterns, p)

	if p.Flags&SingleMatch > 0 {
		ac.hasSingleMatch = true
	}
	ac.size = len(ac.patterns)
	if p.ID > ac.maxID {
		ac.maxID = p.ID
	}
	return nil
}

func (ac *ACKS) Build() {
	ac.initTranslateTable()
	ac.buildStateMachine()
}

func (ac *ACKS) initTranslateTable() {
	var counts [256]int

	// 1. Count occurrences, merging uppercase to lowercase to compress alphabet
	for _, p := range ac.patterns {
		for _, b := range p.Content {
			counts[toLower(b)]++
		}
	}

	// 2. Build translation table
	ac.alphabetSize = 1 // 0 is reserved for unused chars
	for i := 0; i < 256; i++ {
		// Skip uppercase, they will be mapped to lowercase indices later
		if i >= 'A' && i <= 'Z' {
			continue
		}

		if counts[i] > 0 {
			ac.translateTable[i] = uint8(ac.alphabetSize)
			ac.alphabetSize++
		} else {
			ac.translateTable[i] = 0
		}
	}

	// 3. Map uppercase to the same index as lowercase
	for i := 'A'; i <= 'Z'; i++ {
		ac.translateTable[i] = ac.translateTable[i+32]
	}
}

func (ac *ACKS) buildStateMachine() {
	// Temporary Trie structure
	trie := make(map[int]map[uint8]int)
	ac.stateCount = 1 // State 0 is root

	// Initialize output table for state 0
	ac.outputTable = make([][]int, 0)
	ac.outputTable = append(ac.outputTable, []int{})

	// 1. Build Trie (Goto)
	for k, p := range ac.patterns {
		currentState := 0
		for _, b := range p.Content {
			// Use the compressed character code
			tc := ac.translateTable[toLower(b)]

			if trie[currentState] == nil {
				trie[currentState] = make(map[uint8]int)
			}

			if next, exists := trie[currentState][tc]; exists {
				currentState = next
			} else {
				newState := ac.stateCount
				ac.stateCount++
				trie[currentState][tc] = newState
				// Expand output table
				ac.outputTable = append(ac.outputTable, []int{})
				currentState = newState
			}
		}
		ac.outputTable[currentState] = append(ac.outputTable[currentState], k)
	}

	// 2. Build Failure Table
	failure := make([]int, ac.stateCount)
	queue := []int{}

	// Depth 1 failure links point to root (0)
	if rootTrans, ok := trie[0]; ok {
		for _, nextState := range rootTrans {
			queue = append(queue, nextState)
			failure[nextState] = 0
		}
	}

	// BFS
	for len(queue) > 0 {
		rState := queue[0]
		queue = queue[1:]

		if transitions, ok := trie[rState]; ok {
			for charCode, nextState := range transitions {
				queue = append(queue, nextState)
				fState := failure[rState]

				for {
					if trans, ok := trie[fState]; ok {
						if val, ok := trans[charCode]; ok {
							failure[nextState] = val
							break
						}
					}
					if fState == 0 {
						failure[nextState] = 0
						break
					}
					fState = failure[fState]
				}
				// Merge outputs
				ac.outputTable[nextState] = append(ac.outputTable[nextState], ac.outputTable[failure[nextState]]...)
			}
		}
	}

	// 3. Build Delta Table (State Table)
	ac.stateTable = make([]int32, ac.stateCount*ac.alphabetSize)

	for state := 0; state < ac.stateCount; state++ {
		for charIdx := 0; charIdx < ac.alphabetSize; charIdx++ {
			nextState := 0
			curr := state
			found := false

			// Find transition
			for {
				if trans, ok := trie[curr]; ok {
					if val, ok := trans[uint8(charIdx)]; ok {
						nextState = val
						found = true
						break
					}
				}
				if curr == 0 {
					break
				}
				curr = failure[curr]
			}

			if !found {
				nextState = 0
			}

			ac.stateTable[state*ac.alphabetSize+charIdx] = int32(nextState)
		}
	}
}

func (ac *ACKS) Search(text []byte) ([]uint, error) {
	matches := make([]uint, 0, ac.size)
	h := func(pos uint64, ps Pattern) error {
		matches = append(matches, ps.ID)
		return nil
	}
	err := ac.searchPatterns(text, h)
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func (ac *ACKS) Scan(text []byte, m MatchedHandler) error {
	h := func(pos uint64, ps Pattern) error {
		err := m(ps.ID, 0, pos)
		if err != nil {
			return err
		}
		return nil
	}
	err := ac.searchPatterns(text, h)
	if err != nil {
		return err
	}
	return nil
}

func (ac *ACKS) searchPatterns(text []byte, matched matchedPattern) error {
	currentState := 0
	const maxSliceSize = 16 * 1024 * 1024
	useSlice := ac.maxID <= maxSliceSize

	var recordSlice []uint64
	var recordMap map[uint]struct{}

	if ac.hasSingleMatch {
		if useSlice {
			recordSlice = make([]uint64, (ac.maxID/64)+1)
		} else {
			recordMap = make(map[uint]struct{})
		}
	}

	for i, b := range text {
		tc := ac.translateTable[b]

		// O(1) transition
		idx := currentState*ac.alphabetSize + int(tc)
		if idx >= len(ac.stateTable) {
			currentState = 0
		} else {
			currentState = int(ac.stateTable[idx])
		}

		// Check outputs
		if len(ac.outputTable[currentState]) > 0 {
			for _, id := range ac.outputTable[currentState] {
				pat := &ac.patterns[id]
				if pat.Flags&SingleMatch > 0 {
					if useSlice {
						idx := pat.ID / 64
						mask := uint64(1) << (pat.ID % 64)
						if recordSlice[idx]&mask != 0 {
							continue
						}
						recordSlice[idx] |= mask
					} else {
						if _, exists := recordMap[pat.ID]; exists {
							continue
						}
						recordMap[pat.ID] = struct{}{}
					}
				}

				if pat.Flags&Caseless > 0 {
					err := matched(uint64(i+1), *pat)
					if err != nil {
						return err
					}
				} else {
					if memcmp(pat.Content, text[i-pat.strlen+1:], pat.strlen) {
						err := matched(uint64(i+1), *pat)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	return nil
}

func memcmp(a, b []byte, l int) bool {
	if l > len(b) || l > len(a) {
		return false
	}
	return bytes.Equal(a[:l], b[:l])
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}
