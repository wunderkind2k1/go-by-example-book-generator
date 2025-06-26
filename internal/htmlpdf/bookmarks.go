package htmlpdf

import (
	"fmt"
	"log"
	"os"

	"go-by-example-book/internal/github"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ApplyBookmarksParams holds all parameters needed to apply bookmarks to a PDF.
type ApplyBookmarksParams struct {
	TempMergedPDF     string           // Path to the temporary merged PDF file
	FinalPDF          string           // Path where the final PDF with bookmarks should be saved
	Examples          []github.Example // Slice of examples to create bookmarks for
	IntroPageCount    int              // Number of pages in the introduction section
	ExamplePageCounts []int            // Slice containing page counts for each example
}

// ApplyBookmarks adds navigation bookmarks to a PDF file
//
// This function creates a structured bookmark hierarchy for the PDF,
// including an introduction bookmark and individual bookmarks for each
// example with correct page ranges. The bookmarks provide easy navigation
// through the PDF document.
//
// The function handles the case where bookmark creation might fail by
// falling back to simply renaming the temporary file to the final filename.
//
// Parameters:
//   - params: ApplyBookmarksParams struct containing all necessary parameters
//
// Returns:
//   - error: Any error that occurred during bookmark creation
//
// Example:
//
//	err := ApplyBookmarks(ApplyBookmarksParams{...})
//	if err != nil {
//	    log.Fatal(err)
//	}
func ApplyBookmarks(params ApplyBookmarksParams) error {
	fmt.Println("[INFO] Adding bookmarks to PDF...")

	var bookmarks []pdfcpu.Bookmark

	// Add intro bookmark
	bookmarks = append(bookmarks, pdfcpu.Bookmark{
		Title:    "Introduction & Table of Contents",
		PageFrom: 1,
		PageThru: params.IntroPageCount, // Intro and TOC span the actual number of pages
	})

	// Add bookmarks for each example with correct page ranges
	// Examples start after the intro pages
	exampleStartPage := params.IntroPageCount + 1
	for i, ex := range params.Examples {
		pageCount := params.ExamplePageCounts[i]
		bookmarks = append(bookmarks, pdfcpu.Bookmark{
			Title:    fmt.Sprintf("%d. %s", i+1, ex.Title),
			PageFrom: exampleStartPage,
			PageThru: exampleStartPage + pageCount - 1, // -1 because PageThru is inclusive
		})
		exampleStartPage += pageCount // Move to the next example's starting page
	}

	// Add bookmarks to the final PDF
	conf := model.NewDefaultConfiguration()
	err := api.AddBookmarksFile(params.TempMergedPDF, params.FinalPDF, bookmarks, true, conf)
	if err != nil {
		log.Printf("[WARNING] Could not add bookmarks: %v", err)
		// If bookmark creation fails, just copy the temp file
		err = os.Rename(params.TempMergedPDF, params.FinalPDF)
		if err != nil {
			return fmt.Errorf("could not rename temp file: %v", err)
		}
	} else {
		fmt.Println("[BOOKMARKS ADDED] Navigation bookmarks created")
		// Remove the temp file since we created the final one with bookmarks
		os.Remove(params.TempMergedPDF)
	}

	return nil
}
