package netscape

import (
	"fmt"
	"html"
	"io"
	"strings"
	"time"
)

// Bookmark represents a single bookmark entry
type Bookmark struct {
	URL         string
	Title       string
	Description string
	Tags        []string
	AddDate     time.Time
}

// Folder represents a bookmark folder
type Folder struct {
	Name      string
	Bookmarks []Bookmark
	Folders   []Folder
}

// Exporter handles exporting bookmarks to Netscape HTML format
type Exporter struct {
	writer io.Writer
}

// NewExporter creates a new Netscape HTML exporter
func NewExporter(w io.Writer) *Exporter {
	return &Exporter{writer: w}
}

// Export writes bookmarks in Netscape HTML format
func (e *Exporter) Export(bookmarks []Bookmark) error {
	// Write header
	if err := e.writeHeader(); err != nil {
		return err
	}

	// Start bookmark list
	if _, err := fmt.Fprintln(e.writer, "<DL><p>"); err != nil {
		return err
	}

	// Write each bookmark
	for _, bookmark := range bookmarks {
		if err := e.writeBookmark(bookmark); err != nil {
			return err
		}
	}

	// Close bookmark list
	if _, err := fmt.Fprintln(e.writer, "</DL><p>"); err != nil {
		return err
	}

	return nil
}

// ExportWithFolders writes bookmarks organized in folders
func (e *Exporter) ExportWithFolders(folders []Folder, unfiled []Bookmark) error {
	// Write header
	if err := e.writeHeader(); err != nil {
		return err
	}

	// Start main list
	if _, err := fmt.Fprintln(e.writer, "<DL><p>"); err != nil {
		return err
	}

	// Write folders
	for _, folder := range folders {
		if err := e.writeFolder(folder, 1); err != nil {
			return err
		}
	}

	// Write unfiled bookmarks
	if len(unfiled) > 0 {
		// Create an "Unfiled" folder for loose bookmarks
		if _, err := fmt.Fprintln(e.writer, "    <DT><H3>Unfiled</H3>"); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(e.writer, "    <DL><p>"); err != nil {
			return err
		}
		for _, bookmark := range unfiled {
			if err := e.writeBookmark(bookmark); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(e.writer, "    </DL><p>"); err != nil {
			return err
		}
	}

	// Close main list
	if _, err := fmt.Fprintln(e.writer, "</DL><p>"); err != nil {
		return err
	}

	return nil
}

func (e *Exporter) writeHeader() error {
	header := `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<!-- This is an automatically generated file.
     It will be read and overwritten.
     DO NOT EDIT! -->
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>Bookmarks</H1>`

	_, err := fmt.Fprintln(e.writer, header)
	return err
}

func (e *Exporter) writeBookmark(bookmark Bookmark) error {
	indent := "    "

	// Build the bookmark line
	var sb strings.Builder
	sb.WriteString(indent)
	sb.WriteString("<DT><A HREF=\"")
	sb.WriteString(html.EscapeString(bookmark.URL))
	sb.WriteString("\"")

	// Add date if available
	if !bookmark.AddDate.IsZero() {
		sb.WriteString(fmt.Sprintf(" ADD_DATE=\"%d\"", bookmark.AddDate.Unix()))
		// Also add LAST_MODIFIED to match Linkding format
		sb.WriteString(fmt.Sprintf(" LAST_MODIFIED=\"%d\"", bookmark.AddDate.Unix()))
	}

	// Add Linkding-specific attributes to match their export format
	sb.WriteString(" PRIVATE=\"0\"")   // Not private by default
	sb.WriteString(" TOREAD=\"0\"")    // Not marked as "to read" by default

	// Add tags as a TAGS attribute (linkding specific)
	if len(bookmark.Tags) > 0 {
		sb.WriteString(" TAGS=\"")
		sb.WriteString(html.EscapeString(strings.Join(bookmark.Tags, ",")))
		sb.WriteString("\"")
	}

	sb.WriteString(">")
	sb.WriteString(html.EscapeString(bookmark.Title))
	sb.WriteString("</A>")

	// Add description if available
	if bookmark.Description != "" {
		sb.WriteString("\n")
		sb.WriteString(indent)
		sb.WriteString("<DD>")
		sb.WriteString(html.EscapeString(bookmark.Description))
	}

	_, err := fmt.Fprintln(e.writer, sb.String())
	return err
}

func (e *Exporter) writeFolder(folder Folder, depth int) error {
	indent := strings.Repeat("    ", depth)

	// Write folder header
	if _, err := fmt.Fprintf(e.writer, "%s<DT><H3>%s</H3>\n", indent, html.EscapeString(folder.Name)); err != nil {
		return err
	}

	// Start folder content
	if _, err := fmt.Fprintf(e.writer, "%s<DL><p>\n", indent); err != nil {
		return err
	}

	// Write bookmarks in folder
	for _, bookmark := range folder.Bookmarks {
		if err := e.writeBookmark(bookmark); err != nil {
			return err
		}
	}

	// Write nested folders
	for _, subfolder := range folder.Folders {
		if err := e.writeFolder(subfolder, depth+1); err != nil {
			return err
		}
	}

	// Close folder
	if _, err := fmt.Fprintf(e.writer, "%s</DL><p>\n", indent); err != nil {
		return err
	}

	return nil
}