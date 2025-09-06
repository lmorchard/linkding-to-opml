package opml

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"linkding-to-opml/internal/feeds"

	"github.com/sirupsen/logrus"
)

// OPML represents the root OPML document structure
type OPML struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    Head     `xml:"head"`
	Body    Body     `xml:"body"`
}

// Head contains metadata about the OPML document
type Head struct {
	XMLName     xml.Name `xml:"head"`
	Title       string   `xml:"title"`
	DateCreated string   `xml:"dateCreated,omitempty"`
	DateModified string  `xml:"dateModified,omitempty"`
	OwnerName   string   `xml:"ownerName,omitempty"`
	OwnerEmail  string   `xml:"ownerEmail,omitempty"`
	Docs        string   `xml:"docs,omitempty"`
}

// Body contains the outline elements
type Body struct {
	XMLName  xml.Name  `xml:"body"`
	Outlines []Outline `xml:"outline"`
}

// Outline represents a feed entry in the OPML
type Outline struct {
	XMLName xml.Name `xml:"outline"`
	Title   string   `xml:"title,attr"`
	Text    string   `xml:"text,attr"`
	XMLURL  string   `xml:"xmlUrl,attr"`
	HTMLURL string   `xml:"htmlUrl,attr"`
	Type    string   `xml:"type,attr,omitempty"`
}

// GenerateOPML creates an OPML document from feed discovery results
func GenerateOPML(results []*feeds.FeedDiscoveryResult, title string) *OPML {
	logrus.WithFields(logrus.Fields{
		"feed_count": len(results),
		"title":      title,
	}).Debug("Generating OPML document")

	now := time.Now().Format(time.RFC1123)
	
	opml := &OPML{
		Version: "2.0",
		Head: Head{
			Title:        title,
			DateCreated:  now,
			DateModified: now,
			OwnerName:    "linkding-to-opml",
			Docs:         "http://www.opml.org/spec2",
		},
		Body: Body{
			Outlines: make([]Outline, 0, len(results)),
		},
	}

	// Convert feed discovery results to OPML outlines
	for _, result := range results {
		if result.IsSuccessful() {
			outline := Outline{
				Title:   result.FeedTitle,
				Text:    result.FeedTitle,
				XMLURL:  result.FeedURL,
				HTMLURL: result.URL,
				Type:    "rss", // Default to RSS type for feed readers
			}
			
			opml.Body.Outlines = append(opml.Body.Outlines, outline)
			
			logrus.WithFields(logrus.Fields{
				"feed_title": result.FeedTitle,
				"feed_url":   result.FeedURL,
				"html_url":   result.URL,
			}).Debug("Added feed to OPML")
		}
	}

	logrus.WithField("outline_count", len(opml.Body.Outlines)).Info("Generated OPML document")
	
	return opml
}

// WriteOPML writes an OPML document to a file
func WriteOPML(opml *OPML, filePath string) error {
	logrus.WithField("file_path", filePath).Info("Writing OPML file")

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create OPML file: %w", err)
	}
	defer file.Close()

	// Write XML declaration
	if _, err := file.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`); err != nil {
		return fmt.Errorf("failed to write XML declaration: %w", err)
	}
	if _, err := file.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	// Create XML encoder with indentation
	encoder := xml.NewEncoder(file)
	encoder.Indent("", "  ")

	// Encode OPML structure
	if err := encoder.Encode(opml); err != nil {
		return fmt.Errorf("failed to encode OPML: %w", err)
	}

	// Flush encoder
	if err := encoder.Flush(); err != nil {
		return fmt.Errorf("failed to flush XML encoder: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"file_path":     filePath,
		"outline_count": len(opml.Body.Outlines),
	}).Info("Successfully wrote OPML file")

	return nil
}

// ValidateOPML performs basic validation on an OPML document
func ValidateOPML(opml *OPML) error {
	if opml == nil {
		return fmt.Errorf("OPML document is nil")
	}

	if opml.Version != "2.0" {
		return fmt.Errorf("unsupported OPML version: %s", opml.Version)
	}

	if opml.Head.Title == "" {
		return fmt.Errorf("OPML head title is required")
	}

	if len(opml.Body.Outlines) == 0 {
		logrus.Warn("OPML document has no outlines (no feeds)")
	}

	// Validate outlines
	for i, outline := range opml.Body.Outlines {
		if outline.XMLURL == "" {
			return fmt.Errorf("outline %d is missing xmlUrl attribute", i)
		}
		
		if outline.HTMLURL == "" {
			return fmt.Errorf("outline %d is missing htmlUrl attribute", i)
		}
		
		if outline.Title == "" && outline.Text == "" {
			logrus.WithField("outline_index", i).Warn("Outline has no title or text")
		}
	}

	logrus.WithField("outline_count", len(opml.Body.Outlines)).Debug("OPML validation passed")
	
	return nil
}

// GetStats returns statistics about the OPML document
func (o *OPML) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"version":       o.Version,
		"title":         o.Head.Title,
		"outline_count": len(o.Body.Outlines),
		"date_created":  o.Head.DateCreated,
	}
}