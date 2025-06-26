package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

type Example struct {
	Title   string
	Content string
	File    string
}

func sanitizeFilename(title string) string {
	title = strings.ToLower(strings.TrimSpace(title))
	re := regexp.MustCompile(`[^\w]+`)
	return re.ReplaceAllString(title, "_")
}

// extractWords splits a filename into meaningful words
func extractWords(filename string) []string {
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

// wordOverlap calculates the overlap ratio between two word sets
// Uses Jaccard similarity: intersection / union of the word sets
func wordOverlap(originalWords, existingWords []string) float64 {
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

func downloadAsset(url, filename, outputDir string) error {
	content, err := downloadFile(url)
	if err != nil {
		return err
	}

	filepath := filepath.Join(outputDir, filename)
	return os.WriteFile(filepath, []byte(content), 0644)
}

func getExampleFilesFromGitHub() ([]string, error) {
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

func getGitHubFiles(outputDir string) ([]Example, error) {
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
	exampleFiles, err := getExampleFilesFromGitHub()
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
		originalWords := extractWords(filename)

		// Scan existing HTML files to find a match
		entries, err := os.ReadDir(outputDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".html") {
					// Extract words from the existing HTML filename
					existingWords := extractWords(strings.TrimSuffix(entry.Name(), ".html"))

					// Check if there's significant word overlap
					if wordOverlap(originalWords, existingWords) >= 0.7 { // 70% overlap threshold
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

func createHTMLFile(content, filepath string) error {
	return os.WriteFile(filepath, []byte(content), 0644)
}

func htmlToPDF(browser *rod.Browser, htmlPath, pdfPath string) error {
	// Convert to absolute path for file:// URL
	absPath, err := filepath.Abs(htmlPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
	}

	page := browser.MustPage("file://" + absPath)
	defer page.Close()

	// Wait for content to load
	page.MustWaitStable()

	// Generate PDF with default options
	margin := 0.8 // 20mm in inches
	stream, err := page.PDF(&proto.PagePrintToPDF{
		PrintBackground:   true,
		MarginTop:         &margin,
		MarginBottom:      &margin,
		MarginLeft:        &margin,
		MarginRight:       &margin,
		PreferCSSPageSize: true,
	})
	if err != nil {
		return fmt.Errorf("failed to generate PDF: %v", err)
	}

	// Save the PDF to file
	f, err := os.Create(pdfPath)
	if err != nil {
		return fmt.Errorf("failed to create PDF file: %v", err)
	}
	defer f.Close()

	_, err = io.Copy(f, stream)
	if err != nil {
		return fmt.Errorf("failed to write PDF: %v", err)
	}

	return nil
}

func main() {
	fmt.Println("[INFO] Starting Go by Example PDF generator with Rod + pdfcpu...")
	outputDir := "files"
	os.MkdirAll(outputDir, 0755)

	examples, err := getGitHubFiles(outputDir)
	if err != nil {
		log.Fatalf("[ERROR] Failed to get examples: %v", err)
	}
	fmt.Printf("[INFO] Found %d examples\n", len(examples))

	// Initialize Rod browser
	browser := rod.New().MustConnect()
	defer browser.MustClose()

	// Generate individual PDFs first (without TOC)
	var pdfPaths []string
	var examplePageCounts []int // Track page count for each example

	// Generate individual example PDFs
	for i, ex := range examples {
		htmlPath := filepath.Join(outputDir, ex.File+".html")
		pdfPath := filepath.Join(outputDir, ex.File+".pdf")

		// Check if both HTML and PDF already exist
		htmlExists := false
		pdfExists := false

		if _, err := os.Stat(htmlPath); err == nil {
			htmlExists = true
		}
		if _, err := os.Stat(pdfPath); err == nil {
			pdfExists = true
		}

		// If both files exist, skip this example
		if htmlExists && pdfExists {
			fmt.Printf("[SKIPPED] %s (files already exist)\n", ex.Title)
			pdfPaths = append(pdfPaths, pdfPath)

			// Get page count of existing PDF
			pageCount, err := api.PageCountFile(pdfPath)
			if err != nil {
				log.Printf("[WARNING] Could not get page count for %s: %v", ex.Title, err)
				pageCount = 1 // fallback assumption
			}
			examplePageCounts = append(examplePageCounts, pageCount)
			continue
		}

		// Save original HTML content (only if HTML doesn't exist)
		if !htmlExists {
			err = createHTMLFile(ex.Content, htmlPath)
			if err != nil {
				log.Printf("[ERROR] Could not create HTML for %s: %v", ex.Title, err)
				continue
			}
		}

		// Convert to PDF (only if PDF doesn't exist)
		if !pdfExists {
			err = htmlToPDF(browser, htmlPath, pdfPath)
			if err != nil {
				log.Printf("[ERROR] Could not create PDF for %s: %v", ex.Title, err)
				continue
			}
			fmt.Printf("[PDF CREATED] %s.pdf (Example %d)\n", ex.File, i+1)
		} else {
			fmt.Printf("[PDF EXISTS] %s.pdf (Example %d)\n", ex.File, i+1)
		}

		pdfPaths = append(pdfPaths, pdfPath)

		// Get page count of the generated PDF
		pageCount, err := api.PageCountFile(pdfPath)
		if err != nil {
			log.Printf("[WARNING] Could not get page count for %s: %v", ex.Title, err)
			pageCount = 1 // fallback assumption
		}
		examplePageCounts = append(examplePageCounts, pageCount)
		fmt.Printf("[PAGE COUNT] %s: %d pages\n", ex.Title, pageCount)

		// Small delay to be nice to the browser
		time.Sleep(100 * time.Millisecond)
	}

	// Merge all example PDFs into one (without TOC)
	mergedExamplesPdf := filepath.Join(outputDir, "merged_examples.pdf")

	// Use pdfcpu to merge PDFs
	conf := model.NewDefaultConfiguration()

	err = api.MergeCreateFile(pdfPaths, mergedExamplesPdf, false, conf)
	if err != nil {
		log.Fatalf("[ERROR] Could not merge example PDFs: %v", err)
	}
	fmt.Printf("[EXAMPLES MERGED] %s\n", mergedExamplesPdf)

	// Create intro page with TOC and instructions
	fmt.Println("[INFO] Creating intro page...")

	// First, create a temporary TOC with placeholder page numbers
	tempIntroHTML := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Go by Example - Introduction</title>
    <link rel="stylesheet" href="site.css">
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 30px;
            line-height: 1.6;
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #333;
            padding-bottom: 8px;
            font-size: 24px;
            margin-bottom: 20px;
        }
        h2 {
            color: #555;
            font-size: 18px;
            margin-bottom: 15px;
        }
        .intro {
            background-color: #f8f9fa;
            border-left: 4px solid #0066cc;
            padding: 15px;
            margin-bottom: 20px;
            border-radius: 4px;
        }
        .intro h3 {
            color: #0066cc;
            margin-top: 0;
            font-size: 16px;
        }
        .toc-container {
            font-size: 14px;
            line-height: 1.4;
        }
        .toc-container ul {
            font-size: 14px;
        }
        .toc-container li {
            margin-bottom: 6px;
            line-height: 1.3;
        }
        .page-number {
            color: #666;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <h1>Go by Example as a E-Book</h1>
    <h2>Famously published at https://gobyexample.com</h2>

    <div class="intro">
        <h3>📖 Navigation</h3>
        <p>Use your PDF viewer's bookmark panel to navigate between examples. The bookmarks provide clickable links to jump directly to each Go programming example.</p>
    </div>

    <div style="page-break-before: always;"></div>

    <h2>Table of Contents</h2>
    <div class="toc-container">
        <ul>
`

	// Add placeholder TOC entries
	for i, ex := range examples {
		tempIntroHTML += fmt.Sprintf("        <li><span class=\"page-number\">Page %d:</span> %s</li>\n", i+1, ex.Title)
	}

	tempIntroHTML += `        </ul>
    </div>

    <script src="site.js"></script>
</body>
</html>`

	tempIntroHtmlPath := filepath.Join(outputDir, "temp_intro.html")
	err = createHTMLFile(tempIntroHTML, tempIntroHtmlPath)
	if err != nil {
		log.Fatalf("[ERROR] Could not create temp intro HTML: %v", err)
	}

	tempIntroPdfPath := filepath.Join(outputDir, "temp_intro.pdf")
	err = htmlToPDF(browser, tempIntroHtmlPath, tempIntroPdfPath)
	if err != nil {
		log.Fatalf("[ERROR] Could not create temp intro PDF: %v", err)
	}

	// Get the actual page count of the intro PDF
	introPageCount, err := api.PageCountFile(tempIntroPdfPath)
	if err != nil {
		log.Printf("[WARNING] Could not get intro page count: %v", err)
		introPageCount = 2 // fallback assumption
	}
	fmt.Printf("[INTRO PAGE COUNT] %d pages\n", introPageCount)

	// Now create the final intro HTML with correct page numbers
	introHTML := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Go by Example - Introduction</title>
    <link rel="stylesheet" href="site.css">
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 30px;
            line-height: 1.6;
        }
        h1 {
            color: #333;
            border-bottom: 2px solid #333;
            padding-bottom: 8px;
            font-size: 24px;
            margin-bottom: 20px;
        }
        h2 {
            color: #555;
            font-size: 18px;
            margin-bottom: 15px;
        }
        .intro {
            background-color: #f8f9fa;
            border-left: 4px solid #0066cc;
            padding: 15px;
            margin-bottom: 20px;
            border-radius: 4px;
        }
        .intro h3 {
            color: #0066cc;
            margin-top: 0;
            font-size: 16px;
        }
        .toc-container {
            font-size: 14px;
            line-height: 1.4;
        }
        .toc-container ul {
            font-size: 14px;
        }
        .toc-container li {
            margin-bottom: 6px;
            line-height: 1.3;
        }
        .page-number {
            color: #666;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <h1>Go by Example as a E-Book</h1>
    <h2>Famously published at https://gobyexample.com</h2>

    <div class="intro">
        <h3>📖 Navigation</h3>
        <p>Use your PDF viewer's bookmark panel to navigate between examples. The bookmarks provide clickable links to jump directly to each Go programming example.</p>
    </div>

    <div style="page-break-before: always;"></div>

    <h2>Table of Contents</h2>
    <div class="toc-container">
        <ul>
`

	// Calculate correct page numbers for TOC
	// Examples start after the intro pages
	currentPage := introPageCount + 1
	for i, ex := range examples {
		introHTML += fmt.Sprintf("        <li><span class=\"page-number\">Page %d:</span> %s</li>\n", currentPage, ex.Title)
		currentPage += examplePageCounts[i] // Add the actual page count for this example
	}

	introHTML += `        </ul>
    </div>

    <script src="site.js"></script>
</body>
</html>`

	introHtmlPath := filepath.Join(outputDir, "intro.html")
	err = createHTMLFile(introHTML, introHtmlPath)
	if err != nil {
		log.Fatalf("[ERROR] Could not create intro HTML: %v", err)
	}

	introPdfPath := filepath.Join(outputDir, "intro.pdf")
	err = htmlToPDF(browser, introHtmlPath, introPdfPath)
	if err != nil {
		log.Fatalf("[ERROR] Could not create intro PDF: %v", err)
	}
	fmt.Printf("[INTRO PDF CREATED] intro.pdf\n")

	// Clean up temporary files
	os.Remove(tempIntroHtmlPath)
	os.Remove(tempIntroPdfPath)

	// Now merge intro with examples
	tempMergedPdf := filepath.Join(outputDir, "temp_with_intro.pdf")
	introAndExamples := []string{introPdfPath, mergedExamplesPdf}

	err = api.MergeCreateFile(introAndExamples, tempMergedPdf, false, conf)
	if err != nil {
		log.Fatalf("[ERROR] Could not merge intro with examples: %v", err)
	}

	// Add bookmarks to the final PDF
	fmt.Println("[INFO] Adding bookmarks to PDF...")

	var bookmarks []pdfcpu.Bookmark

	// Add intro bookmark
	bookmarks = append(bookmarks, pdfcpu.Bookmark{
		Title:    "Introduction & Table of Contents",
		PageFrom: 1,
		PageThru: introPageCount, // Intro and TOC span the actual number of pages
	})

	// Add bookmarks for each example with correct page ranges
	// Examples start after the intro pages
	exampleStartPage := introPageCount + 1
	for i, ex := range examples {
		pageCount := examplePageCounts[i]
		bookmarks = append(bookmarks, pdfcpu.Bookmark{
			Title:    fmt.Sprintf("%d. %s", i+1, ex.Title),
			PageFrom: exampleStartPage,
			PageThru: exampleStartPage + pageCount - 1, // -1 because PageThru is inclusive
		})
		exampleStartPage += pageCount // Move to the next example's starting page
	}

	// Add bookmarks to the final PDF
	finalPdf := "go_by_example_complete.pdf"
	err = api.AddBookmarksFile(tempMergedPdf, finalPdf, bookmarks, true, conf)
	if err != nil {
		log.Printf("[WARNING] Could not add bookmarks: %v", err)
		// If bookmark creation fails, just copy the temp file
		err = os.Rename(tempMergedPdf, finalPdf)
		if err != nil {
			log.Fatalf("[ERROR] Could not rename temp file: %v", err)
		}
	} else {
		fmt.Println("[BOOKMARKS ADDED] Navigation bookmarks created")
		// Remove the temp file since we created the final one with bookmarks
		os.Remove(tempMergedPdf)
	}

	// Clean up temporary files
	os.Remove(mergedExamplesPdf)
	os.Remove(introPdfPath)
	os.Remove(introHtmlPath)

	fmt.Printf("[COMBINED PDF CREATED] %s\n", finalPdf)
	fmt.Println("[SUCCESS] PDF generation completed!")
	fmt.Printf("[INFO] Individual PDFs saved in: %s/\n", outputDir)
	fmt.Printf("[INFO] Combined PDF saved as: %s\n", finalPdf)
	fmt.Println("[INFO] Use the bookmarks panel in your PDF viewer for navigation!")
}
