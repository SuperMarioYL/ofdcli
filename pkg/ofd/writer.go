package ofd

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
)

// This file is the OFD writer: it projects the agent-callable OfdDocument
// model back onto the GB/T 33190 XML member chain (OFD.xml -> Document.xml ->
// Content.xml) and re-packages them into a compliant OFD zip.
//
// v0.1 scope is a faithful structural round-trip of the modelled tree (text,
// path, image objects + boundaries). Template-driven generation from arbitrary
// JSON input and full visual-layout fidelity are deferred (see roadmap);
// `ofdcli write` currently writes the document model it is handed, which is
// enough to round-trip a read document and to author a model programmatically.

// Write serialises a document model into an OFD container at outPath.
func Write(outPath string, doc *OfdDocument) error {
	if doc == nil {
		return fmt.Errorf("ofd: write: nil document")
	}
	if outPath == "" {
		return fmt.Errorf("ofd: write: empty output path")
	}

	docRoot := doc.DocRoot
	if docRoot == "" {
		docRoot = "Doc_0/Document.xml"
	}
	dir := docDir(docRoot)

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("ofd: create %q: %w", outPath, err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	if err := writeOFDManifest(zw, doc, docRoot); err != nil {
		return err
	}
	if err := writeDocumentXML(zw, doc, docRoot); err != nil {
		return err
	}
	for i := range doc.Pages {
		if err := writePageContent(zw, dir, &doc.Pages[i]); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("ofd: close archive: %w", err)
	}
	return nil
}

// writeOFDManifest writes OFD.xml — the container manifest.
func writeOFDManifest(zw *zip.Writer, doc *OfdDocument, docRoot string) error {
	m := ofdXML{
		DocType: nonEmpty(doc.DocType, "OFD"),
		Version: nonEmpty(doc.Version, "1.1"),
		DocBody: []docBodyXML{{
			DocRoot:         docRoot,
			AnnotationsPage: doc.DocBody.AnnotationsPage,
			DocInfo: &docInfoXML{
				DocID:        doc.DocBody.DocInfo.DocID,
				Title:        doc.DocBody.DocInfo.Title,
				Author:       doc.DocBody.DocInfo.Author,
				Creator:      doc.DocBody.DocInfo.Creator,
				CreationDate: doc.DocBody.DocInfo.CreationDate,
				ModDate:      doc.DocBody.DocInfo.ModDate,
			},
		}},
	}
	return writeXMLMember(zw, "OFD.xml", &m)
}

// writeDocumentXML writes Doc_N/Document.xml — the per-document structural root
// with the page list (each pointing at its Content.xml).
func writeDocumentXML(zw *zip.Writer, doc *OfdDocument, docRoot string) error {
	dx := documentXML{
		Common: &commonXML{
			PageArea: &pageAreaXML{
				PhysicalBox: formatBox(doc.Common.PhysicalBox),
			},
		},
		Pages: make([]pageRefXML, 0, len(doc.Pages)),
	}
	for i := range doc.Pages {
		p := &doc.Pages[i]
		baseLoc := p.BaseLoc
		if baseLoc == "" {
			baseLoc = path.Join("Pages", pageID(i), "Content.xml")
		}
		// Strip the Doc dir prefix if the caller stored a full path in BaseLoc,
		// so the manifest stays consistent with the writer's content-member layout.
		baseLoc = relToDocDir(baseLoc, docRoot)
		dx.Pages = append(dx.Pages, pageRefXML{
			ID:      nonEmpty(p.ID, pageID(i)),
			BaseLoc: baseLoc,
		})
	}
	return writeXMLMember(zw, docRoot, &dx)
}

// writePageContent writes one page's Content.xml at <docDir>/<BaseLoc>.
func writePageContent(zw *zip.Writer, dir string, p *Page) error {
	contentPath := p.BaseLoc
	if contentPath == "" {
		contentPath = path.Join("Pages", p.ID, "Content.xml")
	}
	contentPath = path.Join(dir, relToDocDir(contentPath, dir+"Document.xml"))

	px := pageXML{ID: p.ID}
	if p.Area != nil {
		px.Area = &areaXML{Boundary: formatBox(*p.Area)}
	}
	px.Content = &contentXML{Layers: make([]layerXML, 0, len(p.Layers))}
	for _, l := range p.Layers {
		px.Content.Layers = append(px.Content.Layers, layerToXML(l))
	}
	return writeXMLMember(zw, contentPath, &px)
}

// layerToXML maps a domain Layer to its serialisation form.
func layerToXML(l Layer) layerXML {
	lx := layerXML{ID: l.ID, Type: nonEmpty(l.Type, "Body")}
	lx.TextObjects = make([]textObjectXML, 0, len(l.TextObjects))
	for _, t := range l.TextObjects {
		lx.TextObjects = append(lx.TextObjects, textObjectXML{
			ID:       t.ID,
			Boundary: formatBox(t.Box),
			Font:     t.Font,
			Size:     t.Size,
			TextCodes: []textCodeXML{{Content: t.Text}},
		})
	}
	lx.PathObjects = make([]pathObjectXML, 0, len(l.PathObjects))
	for _, p := range l.PathObjects {
		lx.PathObjects = append(lx.PathObjects, pathObjectXML{
			ID:       p.ID,
			Boundary: formatBox(p.Box),
			Points:   p.Points,
		})
	}
	lx.ImageObjects = make([]imageObjectXML, 0, len(l.ImageObjects))
	for _, im := range l.ImageObjects {
		lx.ImageObjects = append(lx.ImageObjects, imageObjectXML{
			ID:         im.ID,
			Boundary:   formatBox(im.Box),
			ResourceID: im.ResourceID,
		})
	}
	return lx
}

// writeXMLMember marshals v as XML and writes it as a zip member.
func writeXMLMember(zw *zip.Writer, name string, v interface{}) error {
	w, err := zw.Create(name)
	if err != nil {
		return fmt.Errorf("ofd: create member %q: %w", name, err)
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return fmt.Errorf("ofd: write %q header: %w", name, err)
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(v); err != nil {
		return fmt.Errorf("ofd: encode %q: %w", name, err)
	}
	return nil
}

// pageID returns a stable page ID like "Page_0" for the given index.
func pageID(i int) string { return "Page_" + strconv.Itoa(i) }

// relToDocDir strips a leading Doc_N/ prefix from a path so the value stored
// in Document.xml's BaseLoc is relative to the Doc directory (as GB/T 33190
// requires), even when a caller hands in a full member path.
func relToDocDir(member, docRoot string) string {
	dir := docDir(docRoot)
	if dir != "" && len(member) > len(dir) && member[:len(dir)] == dir {
		return member[len(dir):]
	}
	return member
}

func nonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
