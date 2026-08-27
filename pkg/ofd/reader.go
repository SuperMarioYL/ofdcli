package ofd

import (
	"encoding/xml"
	"fmt"
	"path"
	"strings"
)

// This file is the OFD reader: it walks the GB/T 33190 member chain
// OFD.xml -> Doc_N/Document.xml -> Doc_N/Pages/Page_N/Content.xml and projects
// it onto the agent-callable OfdDocument model defined in document.go.
//
// The chain is walked lazily from the manifest outward: OFD.xml gives the
// DocRoot pointer, Document.xml lists pages, and each page's Content.xml is
// read to materialise the object tree.

// Read parses an open OFD container into a document model.
func (c *Container) Read() (*OfdDocument, error) {
	manifest, err := c.readManifest()
	if err != nil {
		return nil, err
	}
	if len(manifest.DocBody) == 0 {
		return nil, fmt.Errorf("ofd: OFD.xml has no DocBody")
	}
	body := manifest.DocBody[0]
	docRoot := body.DocRoot
	if docRoot == "" {
		return nil, fmt.Errorf("ofd: DocBody has no DocRoot")
	}

	docXML, err := c.readDocument(docRoot)
	if err != nil {
		return nil, err
	}

	dir := docDir(docRoot)

	doc := &OfdDocument{
		Version: manifest.Version,
		DocType: manifest.DocType,
		DocRoot: docRoot,
		DocBody: DocBody{
			DocRoot:         docRoot,
			AnnotationsPage: body.AnnotationsPage,
		},
	}
	if body.DocInfo != nil {
		doc.DocBody.DocInfo = DocInfo{
			DocID:        body.DocInfo.DocID,
			Title:        body.DocInfo.Title,
			Author:       body.DocInfo.Author,
			Creator:      body.DocInfo.Creator,
			CreationDate: body.DocInfo.CreationDate,
			ModDate:      body.DocInfo.ModDate,
		}
	}
	if docXML.Common != nil && docXML.Common.PageArea != nil {
		box, err := parseBox(docXML.Common.PageArea.PhysicalBox)
		if err == nil {
			doc.Common.PhysicalBox = box
		}
	}

	pages := make([]Page, 0, len(docXML.Pages))
	for _, ref := range docXML.Pages {
		page, err := c.readPage(dir, ref)
		if err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}
	doc.Pages = pages
	return doc, nil
}

// readManifest unmarshals the OFD.xml container manifest.
func (c *Container) readManifest() (*ofdXML, error) {
	raw, err := c.readMember("OFD.xml")
	if err != nil {
		return nil, fmt.Errorf("ofd: read OFD.xml: %w", err)
	}
	var m ofdXML
	if err := xml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("ofd: parse OFD.xml: %w", err)
	}
	return &m, nil
}

// readDocument unmarshals the Doc_N/Document.xml member.
func (c *Container) readDocument(docRoot string) (*documentXML, error) {
	raw, err := c.readMember(docRoot)
	if err != nil {
		return nil, fmt.Errorf("read Document.xml: %w", err)
	}
	var d documentXML
	if err := xml.Unmarshal(raw, &d); err != nil {
		return nil, fmt.Errorf("parse %s: %w", docRoot, err)
	}
	return &d, nil
}

// readPage reads and maps one page: its Content.xml is resolved relative to the
// enclosing Doc directory via the page ref's BaseLoc.
func (c *Container) readPage(dir string, ref pageRefXML) (Page, error) {
	contentPath := ref.BaseLoc
	if contentPath == "" {
		// No content member; emit an empty page shell.
		return buildPage(ref, &pageXML{})
	}
	contentPath = path.Join(dir, contentPath)
	raw, err := c.readMember(contentPath)
	if err != nil {
		return Page{}, fmt.Errorf("read page %q content %q: %w", ref.ID, contentPath, err)
	}
	var px pageXML
	if err := xml.Unmarshal(raw, &px); err != nil {
		return Page{}, fmt.Errorf("parse %s: %w", contentPath, err)
	}
	return buildPage(ref, &px)
}

// ErrNotOFD is returned when a file is not a recognisable OFD container.
type ErrNotOFD struct{ Path string }

func (e *ErrNotOFD) Error() string {
	return fmt.Sprintf("ofd: %q is not an OFD container (missing OFD.xml)", e.Path)
}

// IsOFD reports whether an open container exposes the OFD.xml manifest member.
func (c *Container) IsOFD() bool {
	for _, n := range c.MemberNames() {
		if strings.EqualFold(n, "OFD.xml") {
			return true
		}
	}
	return false
}
