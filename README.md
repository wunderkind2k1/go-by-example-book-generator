# Go by Example PDF Generator

This project downloads all Go programming language examples from the [Go by Example](https://gobyexample.com) website and converts them to PDF format. It creates both individual PDFs for each example and a combined PDF with a table of contents and navigation bookmarks.

## Features

- **Dynamic example discovery**: Automatically fetches all available examples from the [gobyexample repository](https://github.com/mmcgrana/gobyexample) using GitHub's embedded JSON data
- **Individual PDFs**: Creates a separate PDF for each Go example
- **Combined PDF**: Merges all examples into a single PDF with an introduction page and table of contents
- **Navigation bookmarks**: Includes clickable bookmarks in the PDF for easy navigation between examples
- **Smart caching**: Skips downloading and converting files that already exist, making subsequent runs much faster
- **Dynamic page calculation**: Automatically determines correct page numbers for bookmarks regardless of content length
- **Modern PDF generation**: Uses Rod (headless browser) and pdfcpu for high-quality PDF output
- **Clean formatting**: Applies consistent styling and layout to all PDFs
- **Self-contained**: Downloads all required assets (CSS, JS, images) locally

## Requirements

- Go 1.24.4 or later
- Internet connection (to download examples from GitHub)

## Installation

1. Clone or download this repository
2. Install dependencies:
   ```bash
   go mod tidy
   ```

## Usage

Run the program:
```bash
go run main.go
```

Or build and run:
```bash
go build
./go-by-example-book
```

## Output

The program creates:

- **Combined PDF**: `go_by_example_complete.pdf` in the project root - contains all examples with an introduction page and navigation bookmarks
- **Files directory**: Contains individual PDFs, HTML files, and downloaded assets:
  - **Individual PDFs**: One PDF file per Go example (e.g., `hello_world.pdf`, `functions.pdf`)
  - **HTML files**: Temporary HTML files used for PDF generation
  - **Assets**: Downloaded CSS, JS, and image files

## How it works

1. **Dynamic discovery**: Fetches the list of all available examples from GitHub's embedded JSON data
2. **Smart caching**: Checks for existing HTML and PDF files to avoid redundant work
3. **Asset download**: Downloads all required CSS, JS, and image files from the repository
4. **Content extraction**: Downloads and processes each HTML example file (skipping if already exists)
5. **PDF generation**: Uses Rod (headless Chrome) to render HTML to PDF with proper styling
6. **PDF merging**: Uses pdfcpu to combine all PDFs into a single file
7. **Dynamic bookmark creation**: Calculates actual page numbers and adds navigation bookmarks to the final PDF

## Navigation

The generated PDF includes:

- **Introduction page**: Contains title, subtitle, and navigation instructions
- **Table of contents**: Lists all examples with page numbers (starts on a new page)
- **PDF bookmarks**: Clickable bookmarks in the PDF viewer's navigation panel that jump directly to each example

To navigate the PDF:
- Use your PDF viewer's **Table of Contents** feature (usually accessible via a TOC icon or menu)
- Use the **Bookmarks panel** (most PDF viewers have a bookmarks sidebar)
- Use keyboard shortcuts like `Ctrl/Cmd + G` to jump to specific pages

## Performance

- **First run**: Downloads all examples and creates PDFs (may take several minutes)
- **Subsequent runs**: Only processes new or missing examples, making it much faster
- **Resume capability**: If interrupted, restart the program and it will continue from where it left off

## Dependencies

- `github.com/go-rod/rod` - Headless browser automation for HTML to PDF conversion
- `github.com/pdfcpu/pdfcpu` - PDF processing and merging

## Troubleshooting

### No examples found
- Check your internet connection
- Verify the GitHub repository is accessible
- The program includes debug output to help identify issues

### PDF generation errors
- Rod automatically downloads Chromium on first run (may take a few minutes)
- Ensure you have sufficient disk space for temporary files

### File path issues
- The program uses absolute paths for file:// URLs
- Ensure the output directory is writable

## Example Output Structure

```
go_by_example_complete.pdf        # Combined PDF with bookmarks (project root)
files/
├── hello_world.pdf              # Individual example
├── functions.pdf                # Individual example
├── variables.pdf                # Individual example
├── ...                          # More examples
├── site.css                     # Downloaded assets
├── site.js
├── play.png
└── clipboard.png
```

## Customization

You can modify the templates in `main.go` to change the styling and layout of the generated PDFs. The program uses Go templates for both individual examples and the introduction page.

## Future-proof

The program dynamically discovers all available examples from the GitHub repository, so it will automatically include any new examples added to the Go by Example project without requiring code updates.

## License

This project is open source. Feel free to modify and distribute as needed.
