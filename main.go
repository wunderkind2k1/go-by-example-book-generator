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

func extractTitleFromHTML(htmlContent string) string {
	titleRegex := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`)
	matches := titleRegex.FindStringSubmatch(htmlContent)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
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
	fmt.Printf("[INFO] Fetching %d examples from GitHub...\n", len(exampleFiles))

	for _, filename := range exampleFiles {
		url := fmt.Sprintf("https://raw.githubusercontent.com/mmcgrana/gobyexample/master/public/%s", filename)

		fmt.Printf("[DOWNLOADING] %s\n", filename)

		htmlContent, err := downloadFile(url)
		if err != nil {
			log.Printf("[WARNING] Failed to download %s: %v", filename, err)
			continue
		}

		title := extractTitleFromHTML(htmlContent)
		if title == "" {
			title = filename
		}

		sanitizedFilename := sanitizeFilename(title)

		examples = append(examples, Example{
			Title:   title,
			Content: htmlContent,
			File:    sanitizedFilename,
		})

		fmt.Printf("[DOWNLOADED] %s -> %s\n", title, sanitizedFilename)

		// Small delay to be nice to the server
		time.Sleep(100 * time.Millisecond)
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

	// Generate individual example PDFs
	for i, ex := range examples {
		htmlPath := filepath.Join(outputDir, ex.File+".html")
		pdfPath := filepath.Join(outputDir, ex.File+".pdf")

		// Save original HTML content
		err = createHTMLFile(ex.Content, htmlPath)
		if err != nil {
			log.Printf("[ERROR] Could not create HTML for %s: %v", ex.Title, err)
			continue
		}

		// Convert to PDF
		err = htmlToPDF(browser, htmlPath, pdfPath)
		if err != nil {
			log.Printf("[ERROR] Could not create PDF for %s: %v", ex.Title, err)
			continue
		}

		pdfPaths = append(pdfPaths, pdfPath)
		fmt.Printf("[PDF CREATED] %s.pdf (Example %d)\n", ex.File, i+1)

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
        ul {
            font-size: 16px;
        }
        li {
            margin-bottom: 8px;
            line-height: 1.3;
        }
        .page-number {
            color: #666;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <h1>Go by Example - Complete Book</h1>

    <div class="intro">
        <h3>📖 How to Navigate This Book</h3>
        <p>This PDF contains all Go programming examples with a built-in table of contents. To navigate between examples:</p>
        <ul>
            <li><strong>Use your PDF viewer's Table of Contents feature</strong> - Look for a TOC icon or menu option in your PDF viewer</li>
            <li><strong>Use the Bookmarks panel</strong> - Most PDF viewers have a bookmarks sidebar that shows clickable chapter links</li>
            <li><strong>Keyboard shortcuts</strong> - Many viewers support Ctrl/Cmd + G to jump to specific pages</li>
        </ul>
        <p>The examples below are listed with their page numbers for reference.</p>
    </div>

    <h2>Table of Contents</h2>
    <ul>
`

	for i, ex := range examples {
		pageNum := i + 2 // Intro is page 1, examples start from page 2
		introHTML += fmt.Sprintf("        <li><span class=\"page-number\">Page %d:</span> %d. %s</li>\n", pageNum, i+1, ex.Title)
	}

	introHTML += `    </ul>

    <div class="intro">
        <h3>💡 Tip</h3>
        <p>For the best navigation experience, use your PDF viewer's built-in table of contents rather than trying to click on the page numbers above. The bookmarks in this PDF are fully functional and will take you directly to each example.</p>
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

	// Merge intro with examples
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
		PageThru: 1,
	})

	// Add bookmarks for each example
	// Intro is page 1, examples start from page 2
	for i, ex := range examples {
		bookmarks = append(bookmarks, pdfcpu.Bookmark{
			Title:    fmt.Sprintf("%d. %s", i+1, ex.Title),
			PageFrom: i + 2, // Page 2, 3, 4, etc.
			PageThru: i + 2,
		})
	}

	// Add bookmarks to the final PDF
	finalPdf := filepath.Join(outputDir, "go_by_example_complete.pdf")
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
