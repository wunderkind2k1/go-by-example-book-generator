package main

import (
	"fmt"
	"go-by-example-book/internal/github"
	"go-by-example-book/internal/htmlpdf"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// prepOutputDir prepares the output directory for the PDF generation process
//
// This function creates the output directory if it doesn't exist and returns
// the path to be used throughout the PDF generation process.
//
// Returns:
//   - string: The path to the prepared output directory
func prepOutputDir() string {
	outputDir := "files"
	os.MkdirAll(outputDir, 0755)
	return outputDir
}

// prepHeadlessBrowser initializes and returns a Rod browser instance for PDF generation
//
// This function creates a new headless browser instance that will be used
// for converting HTML files to PDF format. The browser is configured with
// default settings suitable for PDF generation.
//
// Returns:
//   - *rod.Browser: A configured browser instance ready for PDF generation
func prepHeadlessBrowser() *rod.Browser {
	browser := rod.New().MustConnect()
	return browser
}

func main() {
	fmt.Println("[INFO] Starting Go by Example PDF generator with Rod + pdfcpu...")
	outputDir := prepOutputDir()

	examples, err := github.GetGitHubFiles(outputDir)
	if err != nil {
		log.Fatalf("[ERROR] Failed to get examples: %v", err)
	}
	fmt.Printf("[INFO] Found %d examples\n", len(examples))

	browser := prepHeadlessBrowser()
	defer browser.MustClose()

	// Generate individual PDFs first (without TOC)
	var pdfPaths []string
	var examplePageCounts []int // Track page count for each example

	// Generate individual example PDFs
	for i, ex := range examples {
		fileStatus := htmlpdf.ReceiveOutputFileStatus(outputDir, ex.File)

		// If both files exist, skip this example
		if fileStatus.HTMLExists && fileStatus.PDFExists {
			result := htmlpdf.UpdatePageCountForDownloadedExamples(ex, fileStatus, pdfPaths, examplePageCounts)
			pdfPaths = result.PDFPaths
			examplePageCounts = result.ExamplePageCounts
			continue
		}

		// Save original HTML content (only if HTML doesn't exist)
		if !fileStatus.HTMLExists {
			err = htmlpdf.CreateHTMLFile(ex.Content, fileStatus.HTMLPath)
			if err != nil {
				log.Printf("[ERROR] Could not create HTML for %s: %v", ex.Title, err)
				continue
			}
		}

		// Convert to PDF (only if PDF doesn't exist)
		if !fileStatus.PDFExists {
			err = htmlpdf.HTMLToPDF(browser, fileStatus.HTMLPath, fileStatus.PDFPath)
			if err != nil {
				log.Printf("[ERROR] Could not create PDF for %s: %v", ex.Title, err)
				continue
			}
			fmt.Printf("[PDF CREATED] %s.pdf (Example %d)\n", ex.File, i+1)
		} else {
			fmt.Printf("[PDF EXISTS] %s.pdf (Example %d)\n", ex.File, i+1)
		}

		pdfPaths = append(pdfPaths, fileStatus.PDFPath)

		// Get page count of the generated PDF
		pageCount, err := api.PageCountFile(fileStatus.PDFPath)
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
	tempIntroHTML := htmlpdf.CreateBaseHtmlTemplate()

	// Add placeholder TOC entries
	tempIntroHTML += htmlpdf.AddPageInfoToTOC(examples, 1, nil)

	tempIntroHTML += htmlpdf.CloseTOCList()

	tempIntroHtmlPath := filepath.Join(outputDir, "temp_intro.html")
	err = htmlpdf.WriteHTMLAndPDFExp(htmlpdf.HTMLToPDFParams{
		HTMLContent: tempIntroHTML,
		HTMLPath:    tempIntroHtmlPath,
		PDFPath:     filepath.Join(outputDir, "temp_intro.pdf"),
		Browser:     browser,
		Description: "temp intro",
	})
	if err != nil {
		log.Fatalf("[ERROR] Could not create temp intro: %v", err)
	}

	// Get the actual page count of the intro PDF
	introPageCount, err := api.PageCountFile(filepath.Join(outputDir, "temp_intro.pdf"))
	if err != nil {
		log.Printf("[WARNING] Could not get intro page count: %v", err)
		introPageCount = 2 // fallback assumption
	}
	fmt.Printf("[INTRO PAGE COUNT] %d pages\n", introPageCount)

	// Now create the final intro HTML with correct page numbers
	introHTML := htmlpdf.CreateBaseHtmlTemplate()

	// Add TOC entries with correct page numbers
	introHTML += htmlpdf.AddPageInfoToTOC(examples, introPageCount+1, examplePageCounts)

	introHTML += htmlpdf.CloseTOCList()

	introHtmlPath := filepath.Join(outputDir, "intro.html")
	err = htmlpdf.WriteHTMLAndPDFExp(htmlpdf.HTMLToPDFParams{
		HTMLContent: introHTML,
		HTMLPath:    introHtmlPath,
		PDFPath:     filepath.Join(outputDir, "intro.pdf"),
		Browser:     browser,
		Description: "intro",
	})
	if err != nil {
		log.Fatalf("[ERROR] Could not create intro: %v", err)
	}
	fmt.Printf("[INTRO PDF CREATED] intro.pdf\n")

	// Clean up temporary files
	htmlpdf.CleanupTmpFiles(outputDir, []string{"temp_intro.html", "temp_intro.pdf"})

	// Now merge intro with examples
	tempMergedPdf := filepath.Join(outputDir, "temp_with_intro.pdf")
	introAndExamples := []string{filepath.Join(outputDir, "intro.pdf"), mergedExamplesPdf}

	err = api.MergeCreateFile(introAndExamples, tempMergedPdf, false, conf)
	if err != nil {
		log.Fatalf("[ERROR] Could not merge intro with examples: %v", err)
	}

	// Add bookmarks to the final PDF
	fmt.Println("[INFO] Adding bookmarks to PDF...")

	// Add bookmarks to the final PDF
	finalPdf := "go-by-example-generated-ebook.pdf"
	err = htmlpdf.ApplyBookmarks(htmlpdf.ApplyBookmarksParams{
		TempMergedPDF:     tempMergedPdf,
		FinalPDF:          finalPdf,
		Examples:          examples,
		IntroPageCount:    introPageCount,
		ExamplePageCounts: examplePageCounts,
	})
	if err != nil {
		log.Fatalf("[ERROR] Could not apply bookmarks: %v", err)
	}

	// Clean up temporary files
	htmlpdf.CleanupTmpFiles(outputDir, []string{"merged_examples.pdf", "intro.pdf", "intro.html"})

	fmt.Printf("[COMBINED PDF CREATED] %s\n", finalPdf)
	fmt.Println("[SUCCESS] PDF generation completed!")
	fmt.Printf("[INFO] Individual PDFs saved in: %s/\n", outputDir)
	fmt.Printf("[INFO] Combined PDF saved as: %s\n", finalPdf)
	fmt.Println("[INFO] Use the bookmarks panel in your PDF viewer for navigation!")
}
