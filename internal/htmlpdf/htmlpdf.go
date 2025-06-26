// Package htmlpdf provides functionality for creating HTML files and converting
// them to PDF format using the Rod browser automation library.
//
// This package handles the conversion of HTML content to PDF format, which is
// essential for creating printable documents from web content. It uses the Rod
// library to control a headless browser, ensuring that the PDF output accurately
// represents the HTML content including CSS styling and JavaScript-rendered elements.
//
// The package is designed to work with HTML files that reference external assets
// (CSS, JavaScript, images) and handles the conversion process with proper
// margins and formatting for professional document output.
//
// Example usage:
//
//	browser := rod.New().MustConnect()
//	defer browser.MustClose()
//
//	// Create an HTML file
//	err := htmlpdf.CreateHTMLFile("<html><body><h1>Hello World</h1></body></html>", "output.html")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Convert to PDF
//	err = htmlpdf.HTMLToPDF(browser, "output.html", "output.pdf")
//	if err != nil {
//	    log.Fatal(err)
//	}
package htmlpdf

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"go-by-example-book/internal/github"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/pdfcpu/pdfcpu/pkg/api"
)

// CreateHTMLFile creates an HTML file with the given content
//
// This function writes HTML content to a file at the specified path. It's a
// simple wrapper around os.WriteFile that ensures the content is written with
// appropriate file permissions (0644).
//
// The function is commonly used to create temporary HTML files that will be
// converted to PDF, or to save HTML content for later processing.
//
// Parameters:
//   - content: The HTML content to write to the file
//   - filepath: The path where the HTML file should be created
//
// Returns:
//   - error: Any error that occurred during file creation
//
// Example:
//
//	htmlContent := "<html><body><h1>Hello World</h1></body></html>"
//	err := CreateHTMLFile(htmlContent, "example.html")
//	if err != nil {
//	    log.Fatal(err)
//	}
func CreateHTMLFile(content, filepath string) error {
	return os.WriteFile(filepath, []byte(content), 0644)
}

// HTMLToPDF converts an HTML file to PDF using Rod browser
//
// This function performs the conversion of an HTML file to PDF format using
// a headless browser controlled by the Rod library. The process involves:
//
// 1. Converting the HTML file path to an absolute path for the file:// URL
// 2. Loading the HTML file in a browser page
// 3. Waiting for the content to stabilize (CSS and JavaScript execution)
// 4. Generating a PDF with specified formatting options
// 5. Saving the PDF to the specified output path
//
// The function uses professional PDF settings including:
// - Print background enabled for accurate visual representation
// - Consistent margins (0.8 inches) on all sides
// - CSS page size preferences for proper layout
//
// The browser page is automatically closed after the conversion to prevent
// resource leaks.
//
// Parameters:
//   - browser: A Rod browser instance that will be used for the conversion
//   - htmlPath: The path to the input HTML file
//   - pdfPath: The path where the output PDF file should be saved
//
// Returns:
//   - error: Any error that occurred during the conversion process
//
// Example:
//
//	browser := rod.New().MustConnect()
//	defer browser.MustClose()
//
//	err := HTMLToPDF(browser, "input.html", "output.pdf")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Note: The HTML file should be self-contained or reference assets that are
// accessible from the file system. External resources may not load properly
// in the headless browser environment.
func HTMLToPDF(browser *rod.Browser, htmlPath, pdfPath string) error {
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

// FileStatus represents the existence status and paths of HTML and PDF files for an example
type FileStatus struct {
	HTMLExists bool   // Whether the HTML file exists
	PDFExists  bool   // Whether the PDF file exists
	HTMLPath   string // Full path to the HTML file
	PDFPath    string // Full path to the PDF file
}

// PdfData represents the result of processing an example file
type PdfData struct {
	PDFPaths          []string // Updated slice of PDF paths
	ExamplePageCounts []int    // Updated slice of page counts
}

// HTMLToPDFParams contains the parameters for HTML to PDF conversion
type HTMLToPDFParams struct {
	HTMLContent string       // The HTML content to write to the file
	HTMLPath    string       // The path where the HTML file should be created
	PDFPath     string       // The path where the PDF file should be created
	Browser     *rod.Browser // The Rod browser instance to use for PDF conversion
	Description string       // A description of what's being processed (for logging)
}

// ReceiveOutputFileStatus checks if HTML and PDF files already exist for a given example
//
// This function checks the file system to determine if both the HTML and PDF
// files for a specific example already exist in the output directory.
//
// Parameters:
//   - outputDir: The directory where files are stored
//   - filename: The base filename (without extension) for the example
//
// Returns:
//   - FileStatus: A struct containing the existence status and file paths
func ReceiveOutputFileStatus(outputDir, filename string) FileStatus {
	htmlPath := filepath.Join(outputDir, filename+".html")
	pdfPath := filepath.Join(outputDir, filename+".pdf")

	// Check if both HTML and PDF already exist
	htmlExists := false
	pdfExists := false

	if _, err := os.Stat(htmlPath); err == nil {
		htmlExists = true
	}
	if _, err := os.Stat(pdfPath); err == nil {
		pdfExists = true
	}

	return FileStatus{
		HTMLExists: htmlExists,
		PDFExists:  pdfExists,
		HTMLPath:   htmlPath,
		PDFPath:    pdfPath,
	}
}

// UpdatePageCountForDownloadedExamples handles the case when both HTML and PDF files already exist
//
// This function is called when both the HTML and PDF files for an example already exist
// in the output directory. It skips the file generation process and instead:
// 1. Logs that the files are being skipped
// 2. Adds the PDF path to the list of PDFs to merge
// 3. Gets the page count of the existing PDF
// 4. Adds the page count to the tracking slice
//
// Parameters:
//   - ex: The example being processed
//   - fileStatus: The file status information
//   - pdfPaths: Slice to append the PDF path to
//   - examplePageCounts: Slice to append the page count to
//
// Returns:
//   - PdfData: A struct containing the updated PDF paths and page counts
func UpdatePageCountForDownloadedExamples(ex github.Example, fileStatus FileStatus, pdfPaths []string, examplePageCounts []int) PdfData {
	fmt.Printf("[SKIPPED] %s (files already exist)\n", ex.Title)
	pdfPaths = append(pdfPaths, fileStatus.PDFPath)

	// Get page count of existing PDF
	pageCount, err := api.PageCountFile(fileStatus.PDFPath)
	if err != nil {
		log.Printf("[WARNING] Could not get page count for %s: %v", ex.Title, err)
		pageCount = 1 // fallback assumption
	}
	examplePageCounts = append(examplePageCounts, pageCount)

	return PdfData{
		PDFPaths:          pdfPaths,
		ExamplePageCounts: examplePageCounts,
	}
}

// AddPageInfoToTOC adds page information entries to the Table of Contents HTML
//
// This function iterates through the examples and adds formatted list items
// to the HTML Table of Contents with page numbers and example titles.
//
// Parameters:
//   - examples: Slice of examples to add to the TOC
//   - startPage: The starting page number for the examples
//   - examplePageCounts: Slice containing the page count for each example
//
// Returns:
//   - string: The HTML content for the Table of Contents entries
func AddPageInfoToTOC(examples []github.Example, startPage int, examplePageCounts []int) string {
	var tocContent string
	currentPage := startPage

	for i, ex := range examples {
		tocContent += fmt.Sprintf("        <li><span class=\"page-number\"><a href=\"#page=%d\">Page %d</a>:</span> %s</li>\n", currentPage, currentPage, ex.Title)
		if examplePageCounts != nil && i < len(examplePageCounts) {
			currentPage += examplePageCounts[i] // Add the actual page count for this example
		} else {
			currentPage++ // For placeholder TOC, just increment by 1
		}
	}

	return tocContent
}

// CloseTOCList returns the HTML content to close the Table of Contents list
//
// This function provides the closing HTML tags for the TOC list, including
// the closing tags for the list, container div, and body/html elements.
//
// Returns:
//   - string: The HTML content to close the TOC list
func CloseTOCList() string {
	return `        </ul>
    </div>
</body>
</html>`
}

// WriteHTMLAndPDFExp writes HTML content to a file and converts it to PDF
//
// This function performs the common operation of writing HTML content to a file
// and then converting that HTML file to PDF format using the provided browser.
//
// Parameters:
//   - params: HTMLToPDFParams struct containing all necessary parameters
//
// Returns:
//   - error: Any error that occurred during the process
func WriteHTMLAndPDFExp(params HTMLToPDFParams) error {
	// Write HTML file
	err := CreateHTMLFile(params.HTMLContent, params.HTMLPath)
	if err != nil {
		return fmt.Errorf("could not create %s HTML: %v", params.Description, err)
	}

	// Convert to PDF
	err = HTMLToPDF(params.Browser, params.HTMLPath, params.PDFPath)
	if err != nil {
		return fmt.Errorf("could not create %s PDF: %v", params.Description, err)
	}

	return nil
}

// CleanupTmpFiles removes temporary files from the output directory
//
// This function removes various temporary files that are created during
// the PDF generation process, including temporary HTML files, temporary
// PDFs, and intermediate merged files that are no longer needed.
//
// Parameters:
//   - outputDir: The directory containing the temporary files to clean up
//   - files: A slice of file paths to remove
//
// Example:
//
//	files := []string{"temp_intro.html", "temp_intro.pdf"}
//	CleanupTmpFiles("files", files)
func CleanupTmpFiles(outputDir string, files []string) {
	for _, file := range files {
		filePath := filepath.Join(outputDir, file)
		if err := os.Remove(filePath); err != nil {
			// Log but don't fail - cleanup errors are not critical
			log.Printf("[INFO] Could not remove temp file %s: %v", filePath, err)
		}
	}
}
