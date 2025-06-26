![image](https://github.com/user-attachments/assets/61b08f86-0118-4803-aac6-59fa27adb5cf)

# Go by Example PDF Generator

A tool that automatically generates a comprehensive PDF e-book from the [Go by Example](https://gobyexample.com) website. It downloads all Go programming examples and creates a single, navigable PDF with bookmarks and a table of contents.
This is how it is going to look like:
<p align="center">
  <img src="https://github.com/user-attachments/assets/4e305189-8c5c-4699-b0d9-c689563831d9" alt="How the generated go by example e-book looks like in a pdf viewer" width="650">
</p>


## Background & Motivation

The [Go by Example](https://gobyexample.com) website is an excellent resource for learning Go through annotated example programs. However, reading online can be inconvenient, and the site doesn't provide a downloadable format for offline reading.

This tool solves that problem by:
- Automatically fetching all examples from the [Go by Example repository](https://github.com/mmcgrana/gobyexample)
- Converting them to a single, well-formatted PDF e-book
- Adding navigation bookmarks and a table of contents for easy browsing
- Including proper attribution to the original source

The generated e-book is perfect for offline study, reference, or printing.

## How to Build

**Requirements:**
- Go 1.24.4 or later
- Internet connection

**Build steps:**
```bash
# Clone the repository
git clone <repository-url>
cd go-by-example-book

# Install dependencies
go mod tidy

# Build the executable
go build
```

## How to Use

**Run the generator:**
```bash
# Option 1: Run directly
go run main.go

# Option 2: Use the built executable
./go-by-example-book
```

**What happens:**
1. Downloads all Go examples from GitHub (first run takes several minutes)
2. Converts each example to PDF format
3. Creates a combined e-book with navigation bookmarks
4. Cleans up temporary files

**Smart caching:** Subsequent runs are much faster as the tool skips already downloaded examples.

## Results & Files

**Main output:**
- `go-by-example-generated-ebook.pdf` - Complete e-book with all examples, table of contents, and navigation bookmarks

**Files directory (`files/`):**
- Individual PDF files for each example (e.g., `hello_world.pdf`, `functions.pdf`)
- Downloaded assets (CSS, JS, images) from the original site
- Temporary HTML files used during generation

**Navigation features:**
- **PDF bookmarks**: Use your PDF viewer's bookmark panel to jump between examples
- **Table of contents**: Clickable page numbers for direct navigation
- **Introduction page**: Contains attribution and usage instructions

**Attribution included:**
The e-book properly credits the original [Go by Example](https://gobyexample.com) site and includes information about this generator tool.

## Project Structure

```
go-by-example-book/
├── main.go                    # Main orchestration
├── internal/
│   ├── github/               # GitHub API & example fetching
│   ├── htmlpdf/              # HTML/PDF processing & bookmarks
│   └── naming/               # Filename processing
└── README.md
```

## Dependencies

- `github.com/go-rod/rod` - Headless browser for HTML→PDF conversion
- `github.com/pdfcpu/pdfcpu` - PDF processing and merging

## License

Open source. Free to modify and distribute.
