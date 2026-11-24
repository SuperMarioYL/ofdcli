package ofd

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
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

	// A symlinked destination is written through, not replaced by a regular
	// file: resolve it before the completed archive is renamed onto it.
	if resolved, err := filepath.EvalSymlinks(outPath); err == nil {
		outPath = resolved
	}

	// The source container is still an open zip reader on its own path: creating
	// outPath at that path would truncate the file this call is about to read
	// member data from, destroying the caller's input. Refuse before any file
	// is touched (reproduced at v0.2.0: `write -i X -o X` left X at 0 bytes).
	if src != nil && sameFile(src.Path(), outPath) {
		return fmt.Errorf("ofd: write: input and output are the same file (%q); write to a different path", outPath)
	}

	// Package into a temp file beside the destination and rename it into place
	// only once the archive is complete. Creating outPath directly would leave
	// an existing document at that path truncated to nothing whenever packaging
	// fails (e.g. an unreadable source member), losing the caller's previous
	// output along with the in-flight one.
	tmp, err := os.CreateTemp(filepath.Dir(outPath), ".ofdcli-*.tmp")
	if err != nil {
		return fmt.Errorf("ofd: create temp file for %q: %w", outPath, err)
	}
	tmpName := tmp.Name()
	committed := false
	defer func() {
		if !committed {
			tmp.Close()
			os.Remove(tmpName)
		}
	}()

	zw := zip.NewWriter(tmp)
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
	// os.CreateTemp creates 0600 files; keep an existing destination's mode and
	// fall back to the plain 0644 a regular create would leave.
	mode := os.FileMode(0o644)
	if fi, err := os.Stat(outPath); err == nil {
		mode = fi.Mode().Perm()
	}
	if err := tmp.Chmod(mode); err != nil {
		return fmt.Errorf("ofd: set mode on %q: %w", outPath, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("ofd: close %q: %w", outPath, err)
	}
	if err := os.Rename(tmpName, outPath); err != nil {
		return fmt.Errorf("ofd: rename temp archive to %q: %w", outPath, err)
	}
	committed = true
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
// written to. The page-area default is serialised under the standard's
// CommonData element (GB/T 33190 CT_CommonData) and only when the model
// actually carries one — a fabricated "0 0 0 0" box is never emitted. The
// PublicRes/DocumentRes declarations are serialised alongside it: the resource
// maps themselves are carried over as unmodeled members, so omitting these
// references would leave every ImageObject/Font ResourceID dangling.
func writeDocumentXML(zw *zip.Writer, doc *OfdDocument, docRoot string, refs []pageRefXML) error {
	dx := documentXML{Pages: refs}
	common := &commonXML{}
	hasCommon := false
	if doc.Common.PhysicalBox.W != 0 || doc.Common.PhysicalBox.H != 0 {
		common.PageArea = &pageAreaXML{PhysicalBox: formatBox(doc.Common.PhysicalBox)}
		hasCommon = true
	}
	if ref := doc.Common.PublicRes; ref != nil {
		common.PublicRes = &resRefXML{ID: ref.ID, BaseLoc: relToDocDir(ref.BaseLoc, docRoot)}
		hasCommon = true
	}
	if ref := doc.Common.DocumentRes; ref != nil {
		common.DocumentRes = &resRefXML{ID: ref.ID, BaseLoc: relToDocDir(ref.BaseLoc, docRoot)}
		hasCommon = true
	}
	if hasCommon {
		dx.Common = common
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

// sameFile reports whether two paths refer to the same on-disk file. A path
// that does not exist (yet) is never the same as an existing one; for existing
// paths the os.SameFile inode comparison covers symlinks and differing textual
// forms of one file.
func sameFile(a, b string) bool {
	fa, err1 := os.Stat(a)
	fb, err2 := os.Stat(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return os.SameFile(fa, fb)
}
