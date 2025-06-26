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

// CreateBaseHtmlTemplate creates the base HTML template for the introduction page
//
// This function generates the HTML structure for the introduction page that includes:
// - CSS styling for the page layout
// - Header with title and description
// - Navigation instructions
// - Table of Contents section
//
// The template includes placeholders for dynamic content that will be filled in later.
//
// Returns:
//   - string: The complete HTML template as a string
func CreateBaseHtmlTemplate() string {
	return `<!DOCTYPE html>
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
        .page-number a {
            color: #0066cc;
            text-decoration: none;
        }
        .page-number a:hover {
            text-decoration: underline;
        }
    </style>
</head>
<body>
    <h1>Go by Example as a E-Book</h1>
    <h2>Famously published at https://gobyexample.com</h2>

    <div class="intro">
        <h3>📖 Navigation</h3>
        <p>Use your PDF viewer's bookmark panel to navigate between examples. The bookmarks provide clickable links to jump directly to each Go programming example. You can also click on the page numbers in the Table of Contents below to jump directly to each example.</p>
    </div>

    <div style="page-break-before: always;"></div>

    <h2>Table of Contents</h2>
    <div class="toc-container">
        <ul>
`
}
