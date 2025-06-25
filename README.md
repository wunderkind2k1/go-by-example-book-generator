# Go by Example PDF Generator

This project downloads all Go programming language examples from the [Go by Example](https://gobyexample.com) website and converts them to PDF format. It creates both individual PDFs for each example and a combined PDF with a table of contents.

## Features

- **Downloads examples directly from GitHub**: Uses the GitHub API to fetch HTML files from the [gobyexample repository](https://github.com/mmcgrana/gobyexample)
- **Individual PDFs**: Creates a separate PDF for each Go example
- **Combined PDF**: Merges all examples into a single PDF with a table of contents
- **Modern PDF generation**: Uses Rod (headless browser) and pdfcpu for high-quality PDF output
- **Clean formatting**: Applies consistent styling and layout to all PDFs

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

The program creates an `examples_pdf` directory containing:

- **Individual PDFs**: One PDF file per Go example (e.g., `hello_world.pdf`, `functions.pdf`)
- **Combined PDF**: `go_by_example_complete.pdf` - contains all examples with a table of contents
- **HTML files**: Temporary HTML files used for PDF generation

## How it works

1. **GitHub API**: Fetches all HTML files from the gobyexample repository
2. **Content extraction**: Extracts titles and content from each HTML file
3. **Template rendering**: Uses Go templates to create clean HTML for each example
4. **PDF generation**: Uses Rod (headless Chrome) to render HTML to PDF
5. **PDF merging**: Uses pdfcpu to combine all PDFs into a single file

## Dependencies

- `github.com/google/go-github/v62` - GitHub API client
- `github.com/go-rod/rod` - Headless browser automation
- `github.com/pdfcpu/pdfcpu` - PDF processing and merging

## Troubleshooting

### No examples found
- Check your internet connection
- Verify the GitHub API is accessible
- The program includes debug output to help identify issues

### PDF generation errors
- Rod automatically downloads Chromium on first run (may take a few minutes)
- Ensure you have sufficient disk space for temporary files

### File path issues
- The program uses absolute paths for file:// URLs
- Ensure the output directory is writable

## Example Output Structure

```
examples_pdf/
├── toc.pdf                           # Table of contents
├── hello_world.pdf                   # Individual example
├── functions.pdf                     # Individual example
├── structs.pdf                       # Individual example
├── ...                               # More examples
└── go_by_example_complete.pdf        # Combined PDF
```

## Customization

You can modify the templates in `main.go` to change the styling and layout of the generated PDFs. The program uses Go templates for both individual examples and the table of contents.

## License

This project is open source. Feel free to modify and distribute as needed.
