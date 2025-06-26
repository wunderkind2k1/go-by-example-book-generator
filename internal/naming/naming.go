// Package naming provides utilities for filename processing and word-based matching.
//
// This package contains functions for extracting meaningful words from filenames
// and calculating similarity between different filename representations. It's
// primarily used for matching existing HTML files with downloaded content from
// external sources.
//
// The package uses Jaccard similarity (intersection over union) to determine
// how similar two sets of words are, which helps in identifying corresponding
// files even when they have slightly different naming conventions.
//
// Example usage:
//
//	words := naming.ExtractWords("hello-world-example.html")
//	// Returns: ["hello", "world", "example"]
//
//	similarity := naming.WordOverlap(words1, words2)
//	// Returns: float64 between 0.0 and 1.0
package naming

import "strings"

// ExtractWords splits a filename into meaningful words
//
// This function processes a filename by:
// 1. Removing the .html extension
// 2. Splitting on common separators (hyphens, underscores, spaces, colons)
// 3. Converting to lowercase and trimming whitespace
// 4. Filtering out common words like "go", "by", "example" and empty strings
//
// The result is a slice of meaningful words that can be used for comparison
// and matching purposes.
//
// Example:
//
//	ExtractWords("hello-world-example.html") -> ["hello", "world", "example"]
//	ExtractWords("go_by_example_test") -> ["test"]
func ExtractWords(filename string) []string {
	// Remove file extension
	filename = strings.TrimSuffix(filename, ".html")

	// Split by common separators: hyphens, underscores, spaces, colons
	words := strings.FieldsFunc(filename, func(r rune) bool {
		return r == '-' || r == '_' || r == ' ' || r == ':'
	})

	// Filter out empty strings and common words
	var result []string
	for _, word := range words {
		word = strings.ToLower(strings.TrimSpace(word))
		if word != "" && word != "go" && word != "by" && word != "example" {
			result = append(result, word)
		}
	}

	return result
}

// WordOverlap calculates the overlap ratio between two word sets
//
// This function uses Jaccard similarity to measure how similar two sets of words are.
// The formula is: |A ∩ B| / |A ∪ B| where A and B are the word sets.
//
// The result is a value between 0.0 and 1.0:
// - 0.0 means no words overlap
// - 1.0 means the word sets are identical
// - Values in between indicate partial overlap
//
// This is useful for determining if two filenames refer to the same content
// even when they use different naming conventions.
//
// Example:
//
//	words1 := []string{"hello", "world", "example"}
//	words2 := []string{"hello", "world", "test"}
//	overlap := WordOverlap(words1, words2) // Returns 0.5
func WordOverlap(originalWords, existingWords []string) float64 {
	if len(originalWords) == 0 || len(existingWords) == 0 {
		return 0.0
	}

	// Create sets for efficient lookup
	originalWordSet := make(map[string]bool)
	for _, word := range originalWords {
		originalWordSet[word] = true
	}

	existingWordSet := make(map[string]bool)
	for _, word := range existingWords {
		existingWordSet[word] = true
	}

	// Count overlapping words
	overlappingWords := 0
	for word := range originalWordSet {
		if existingWordSet[word] {
			overlappingWords++
		}
	}

	// Calculate overlap ratio (intersection / union)
	totalUniqueWords := len(originalWordSet) + len(existingWordSet) - overlappingWords
	if totalUniqueWords == 0 {
		return 0.0
	}

	return float64(overlappingWords) / float64(totalUniqueWords)
}
