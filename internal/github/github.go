// Package github provides functionality for interacting with GitHub repositories
// and downloading Go by Example content.
//
// This package handles the communication with GitHub's web interface to fetch
// directory listings and download example files. It includes functionality for:
// - Fetching directory listings from GitHub repositories
// - Downloading individual example files
// - Managing assets (CSS, JS, images) required for the examples
// - Processing and organizing downloaded content
//
// The package is specifically designed to work with the gobyexample repository
// structure and handles the parsing of GitHub's embedded JSON data to extract
// file information.
//
// Example usage:
//
//	examples, err := github.GetGitHubFiles("output_directory")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	for _, example := range examples {
//	    fmt.Printf("Title: %s, File: %s\n", example.Title, example.File)
//	}
package github

import (
	"encoding/json"
	"fmt"
	"go-by-example-book/internal/naming"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Example represents a Go by Example with its title, content, and filename
//
// This struct holds the metadata and content for a single Go programming example.
// It's used throughout the application to represent examples that have been
// downloaded from GitHub or found in existing local files.
type Example struct {
	Title   string // The human-readable title of the example
	Content string // The HTML content of the example
	File    string // The sanitized filename for the example
}

// GetExampleFilesFromGitHub fetches the directory listing from GitHub and extracts example files
//
// This function performs the following operations:
// 1. Makes an HTTP request to the GitHub repository page
// 2. Parses the embedded JSON data that GitHub uses to populate the file browser
// 3. Filters the files to include only example files (excluding assets like CSS, JS, images)
// 4. Returns a sorted list of example filenames
//
// The function handles GitHub's specific HTML structure and embedded JSON format
// to extract file information without requiring API access.
//
// Returns:
//   - []string: A slice of example filenames
//   - error: Any error that occurred during the process
//
// Example:
//
//	files, err := GetExampleFilesFromGitHub()
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Found %d example files\n", len(files))
func GetExampleFilesFromGitHub() ([]string, error) {
	// Fetch the directory listing from GitHub
	url := "https://github.com/mmcgrana/gobyexample/tree/master/public"
	fmt.Printf("[DEBUG] Fetching directory listing from: %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch directory listing: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	content := string(body)

	// Find the embedded JSON block
	jsonStart := strings.Index(content, `<script type="application/json" data-target="react-app.embeddedData">`)
	if jsonStart == -1 {
		return nil, fmt.Errorf("could not find embedded JSON block in GitHub page")
	}
	jsonStart += len(`<script type="application/json" data-target="react-app.embeddedData">`)
	jsonEnd := strings.Index(content[jsonStart:], "</script>")
	if jsonEnd == -1 {
		return nil, fmt.Errorf("could not find end of embedded JSON block in GitHub page")
	}
	jsonStr := content[jsonStart : jsonStart+jsonEnd]

	// Parse the JSON
	var embedded struct {
		Payload struct {
			Tree struct {
				Items []struct {
					Name        string `json:"name"`
					ContentType string `json:"contentType"`
				} `json:"items"`
			} `json:"tree"`
		} `json:"payload"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &embedded); err != nil {
		return nil, fmt.Errorf("failed to parse embedded JSON: %v", err)
	}

	var exampleFiles []string
	for _, item := range embedded.Payload.Tree.Items {
		if item.ContentType == "file" &&
			!strings.HasSuffix(item.Name, ".html") &&
			!strings.HasSuffix(item.Name, ".js") &&
			!strings.HasSuffix(item.Name, ".css") &&
			!strings.HasSuffix(item.Name, ".png") &&
			!strings.HasSuffix(item.Name, ".ico") {
			exampleFiles = append(exampleFiles, item.Name)
		}
	}

	sort.Strings(exampleFiles)
	fmt.Printf("[DEBUG] Found %d example files from embedded JSON.\n", len(exampleFiles))
	return exampleFiles, nil
}

// Helper functions needed by getGitHubFiles

// downloadFile downloads content from a URL and returns it as a string
//
// This is a helper function that performs HTTP GET requests and returns
// the response body as a string. It includes proper error handling for
// HTTP status codes and network errors.
func downloadFile(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// downloadAsset downloads a file from a URL and saves it to the specified directory
//
// This helper function combines downloadFile with file writing functionality.
// It's used to download assets like CSS, JavaScript, and image files that
// are required for the examples to display correctly.
func downloadAsset(url, filename, outputDir string) error {
	content, err := downloadFile(url)
	if err != nil {
		return err
	}

	filepath := filepath.Join(outputDir, filename)
	return os.WriteFile(filepath, []byte(content), 0644)
}

// sanitizeFilename converts a title to a safe filename
//
// This function processes a title string to create a filename-safe version by:
// 1. Converting to lowercase
// 2. Trimming whitespace
// 3. Replacing non-word characters with underscores
//
// This ensures that filenames are consistent and safe for file system operations.
func sanitizeFilename(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	re := regexp.MustCompile(`[^\w]+`)
	return re.ReplaceAllString(title, "_")
}

// GetGitHubFiles downloads assets and fetches all examples from GitHub
//
// This is the main function of the package that orchestrates the entire process
// of downloading Go by Example content. It performs the following steps:
//
// 1. Downloads required assets (CSS, JS, images) from the GitHub repository
// 2. Fetches the list of available example files
// 3. For each example file:
//   - Checks if a corresponding HTML file already exists locally
//   - Uses word-based matching to find existing files with similar names
//   - Downloads the example content if no match is found
//   - Creates Example structs with the content and metadata
//
// The function includes intelligent caching - if an HTML file with a similar
// name already exists, it will use that instead of re-downloading the content.
// This is determined using the naming package's word overlap functionality.
//
// Parameters:
//   - outputDir: The directory where files should be saved
//
// Returns:
//   - []Example: A slice of Example structs containing all the examples
//   - error: Any error that occurred during the process
//
// Example:
//
//	examples, err := GetGitHubFiles("./output")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Processed %d examples\n", len(examples))
func GetGitHubFiles(outputDir string) ([]Example, error) {
	// Download required assets first
	fmt.Println("[INFO] Downloading assets...")

	assets := []struct {
		url      string
		filename string
	}{
		{"https://raw.githubusercontent.com/mmcgrana/gobyexample/master/public/site.css", "site.css"},
		{"https://raw.githubusercontent.com/mmcgrana/gobyexample/master/public/site.js", "site.js"},
		{"https://raw.githubusercontent.com/mmcgrana/gobyexample/master/public/play.png", "play.png"},
		{"https://raw.githubusercontent.com/mmcgrana/gobyexample/master/public/clipboard.png", "clipboard.png"},
	}

	for _, asset := range assets {
		fmt.Printf("[DOWNLOADING] %s\n", asset.filename)
		err := downloadAsset(asset.url, asset.filename, outputDir)
		if err != nil {
			log.Printf("[WARNING] Failed to download %s: %v", asset.filename, err)
		} else {
			fmt.Printf("[DOWNLOADED] %s\n", asset.filename)
		}
	}

	// Dynamically fetch all available examples from GitHub
	exampleFiles, err := GetExampleFilesFromGitHub()
	if err != nil {
		return nil, fmt.Errorf("failed to get example files from GitHub: %v", err)
	}

	var examples []Example
	fmt.Printf("[INFO] Processing %d examples...\n", len(exampleFiles))

	for _, filename := range exampleFiles {
		// First, try to find existing HTML files that might match this example
		// We'll use word-based matching to find corresponding files
		var htmlContent string
		var title string
		var sanitizedFilename string
		var foundExisting bool

		// Extract words from the original filename
		originalWords := naming.ExtractWords(filename)

		// Scan existing HTML files to find a match
		entries, err := os.ReadDir(outputDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
					// Extract words from the existing HTML filename
					existingWords := naming.ExtractWords(strings.TrimSuffix(entry.Name(), ".html"))

					// Check if there's significant word overlap
					if naming.WordOverlap(originalWords, existingWords) >= 0.7 { // 70% overlap threshold
						// Found a match, read the HTML file
						htmlPath := filepath.Join(outputDir, entry.Name())
						content, err := os.ReadFile(htmlPath)
						if err != nil {
							log.Printf("[WARNING] Failed to read existing HTML file %s: %v", entry.Name(), err)
							continue
						}
						htmlContent = string(content)
						title = strings.TrimSuffix(entry.Name(), ".html")
						sanitizedFilename = strings.TrimSuffix(entry.Name(), ".html")
						foundExisting = true
						fmt.Printf("[USING EXISTING] %s (as %s.html)\n", title, sanitizedFilename)
						break
					}
				}
			}
		}

		if !foundExisting {
			// Download HTML content from GitHub
			url := fmt.Sprintf("https://raw.githubusercontent.com/mmcgrana/gobyexample/master/public/%s", filename)
			fmt.Printf("[DOWNLOADING] %s\n", filename)

			htmlContent, err = downloadFile(url)
			if err != nil {
				log.Printf("[WARNING] Failed to download %s: %v", filename, err)
				continue
			}

			// Use the URL filename for both title and sanitized filename
			// This ensures consistency and avoids HTML parsing issues
			title = filename
			sanitizedFilename = sanitizeFilename(filename)
			fmt.Printf("[DOWNLOADED] %s -> %s\n", title, sanitizedFilename)
		}

		examples = append(examples, Example{
			Title:   title,
			Content: htmlContent,
			File:    sanitizedFilename,
		})

		// Small delay to be nice to the server (only when downloading)
		if !foundExisting {
			time.Sleep(100 * time.Millisecond)
		}
	}

	sort.Slice(examples, func(i, j int) bool {
		return examples[i].Title < examples[j].Title
	})

	return examples, nil
}
