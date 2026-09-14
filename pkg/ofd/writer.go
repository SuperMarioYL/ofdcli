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
// Only the modelled members (OFD.xml, Document.xml, page Content.xml files)
// are written; use WritePreserving to also carry unmodeled members from a
// source container.
func Write(outPath string, doc *OfdDocument) error {
	return writeArchive(outPath, doc, nil)
}

// WritePreserving serialises a document model like Write and additionally
// copies every unmodeled archive member from src — Res/ fonts and images,
// attachments, custom parts — verbatim, so a read-then-write round-trip keeps
// resource references (e.g. ImageObject ResourceID) resolvable by reference
// viewers.
func WritePreserving(outPath string, doc *OfdDocument, src *Container) error {
	if src == nil {
		return fmt.Errorf("ofd: write: nil source container")
	}
	return writeArchive(outPath, doc, src)
}

// writeArchive writes doc's modelled members and, when src is non-nil, every
// unmodeled member from src.
func writeArchive(outPath string, doc *OfdDocument, src *Container) error {
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
	refs := pageRefs(doc, docRoot)

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("ofd: create %q: %w", outPath, err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	if err := writeOFDManifest(zw, doc, docRoot); err != nil {
		return err
	}
	if err := writeDocumentXML(zw, doc, docRoot, refs); err != nil {
		return err
	}
	for i := range doc.Pages {
		if err := writePageContent(zw, dir, refs[i], &doc.Pages[i]); err != nil {
			return err
		}
	}
	if src != nil {
		if err := copyUnmodeledMembers(zw, src, modeledMemberNames(dir, docRoot, refs)); err != nil {
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("ofd: close archive: %w", err)
	}
	return nil
}

// pageRefs resolves every page's manifest entry up front: the effective
// Content.xml location (the model's BaseLoc, or the writer's default),
// normalised to be relative to the Doc directory. Both the Document.xml
// manifest and the archive's content members are derived from these refs, so
// the two can never disagree on where a page's content lives.
func pageRefs(doc *OfdDocument, docRoot string) []pageRefXML {
	refs := make([]pageRefXML, 0, len(doc.Pages))
	for i := range doc.Pages {
		p := &doc.Pages[i]
		refs = append(refs, pageRefXML{
			ID:      nonEmpty(p.ID, pageID(i)),
			BaseLoc: relToDocDir(effectiveBaseLoc(p, i), docRoot),
		})
	}
	return refs
}

// effectiveBaseLoc returns the page's content path: the model's BaseLoc when
// set, else the default Pages/<ID>/Content.xml location (with the page index
// standing in for a missing ID).
func effectiveBaseLoc(p *Page, i int) string {
	if p.BaseLoc != "" {
		return p.BaseLoc
	}
	return path.Join("Pages", nonEmpty(p.ID, pageID(i)), "Content.xml")
}

// modeledMemberNames lists the archive members writeArchive itself emits, so
// copyUnmodeledMembers never duplicates them.
func modeledMemberNames(dir, docRoot string, refs []pageRefXML) map[string]bool {
	names := map[string]bool{
		"OFD.xml": true,
		docRoot:   true,
	}
	for _, ref := range refs {
		names[path.Join(dir, ref.BaseLoc)] = true
	}
	return names
}

// copyUnmodeledMembers copies every src member whose name is not in the
// modeled set, preserving the original file header (name, method, mtime).
func copyUnmodeledMembers(zw *zip.Writer, src *Container, modeled map[string]bool) error {
	for _, f := range src.reader.Reader.File {
		if modeled[f.Name] {
			continue
		}
		hdr := f.FileHeader
		w, err := zw.CreateHeader(&hdr)
		if err != nil {
			return fmt.Errorf("ofd: copy member %q: %w", f.Name, err)
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("ofd: open member %q: %w", f.Name, err)
		}
		_, err = io.Copy(w, rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("ofd: copy member %q: %w", f.Name, err)
		}
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
// with the page list (each pointing at its Content.xml). The page refs are the
// ones resolved by pageRefs, i.e. the same locations the content members are
// written to.
func writeDocumentXML(zw *zip.Writer, doc *OfdDocument, docRoot string, refs []pageRefXML) error {
	dx := documentXML{
		Common: &commonXML{
			PageArea: &pageAreaXML{
				PhysicalBox: formatBox(doc.Common.PhysicalBox),
			},
		},
		Pages: refs,
	}
	return writeXMLMember(zw, docRoot, &dx)
}

// writePageContent writes one page's Content.xml at the resolved member path
// (dir + ref.BaseLoc) — exactly the location the manifest's BaseLoc points at.
func writePageContent(zw *zip.Writer, dir string, ref pageRefXML, p *Page) error {
	contentPath := path.Join(dir, ref.BaseLoc)

	px := pageXML{ID: ref.ID}
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
