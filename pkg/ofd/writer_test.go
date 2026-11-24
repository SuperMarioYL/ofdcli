package ofd

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Writer tests — the m2 round-trip milestone (model -> compliant OFD zip).
//
// The contract under test: write the model that the reader produced back into
// a fresh .ofd, then re-read it and assert the agent-callable content (page
// count, IDs, text, boundaries, non-text objects) is preserved. This is the
// "render-in-a-reference-viewer" compliance gate the plan names for m2,
// reduced to a machine-checkable structural round-trip.

func TestWrite_RoundTripText(t *testing.T) {
	src := openSample(t)
	defer src.Close()
	doc, err := src.Read()
	if err != nil {
		t.Fatalf("read source: %v", err)
	}

	out := t.TempDir() + "/out.ofd"
	if err := Write(out, doc); err != nil {
		t.Fatalf("Write: %v", err)
	}

	back, err := Open(out)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer back.Close()
	rd, err := back.Read()
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if rd.PageCount() != doc.PageCount() {
		t.Fatalf("page count round-trip = %d, want %d", rd.PageCount(), doc.PageCount())
	}
	requireEqualText(t, rd)
}

func TestWrite_RoundTripIDsAndBoundaries(t *testing.T) {
	src := openSample(t)
	defer src.Close()
	doc, err := src.Read()
	if err != nil {
		t.Fatalf("read source: %v", err)
	}

	out := t.TempDir() + "/ids.ofd"
	if err := Write(out, doc); err != nil {
		t.Fatalf("Write: %v", err)
	}
	back, err := Open(out)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer back.Close()
	rd, err := back.Read()
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}

	for i, p := range rd.Pages {
		if p.ID != doc.Pages[i].ID {
			t.Fatalf("page %d ID = %q, want %q", i, p.ID, doc.Pages[i].ID)
		}
		if p.Area == nil || p.Area.W != doc.Pages[i].Area.W {
			t.Fatalf("page %d area round-trip mismatch: %+v vs %+v", i, p.Area, doc.Pages[i].Area)
		}
		for j, to := range p.Layers[0].TextObjects {
			want := doc.Pages[i].Layers[0].TextObjects[j]
			if to.ID != want.ID || to.Box != want.Box || to.Size != want.Size || to.Font != want.Font {
				t.Fatalf("page %d text %d round-trip = %+v, want %+v", i, j, to, want)
			}
		}
	}
}

func TestWrite_RoundTripNonTextObjects(t *testing.T) {
	src := openSample(t)
	defer src.Close()
	doc, err := src.Read()
	if err != nil {
		t.Fatalf("read source: %v", err)
	}

	out := t.TempDir() + "/objs.ofd"
	if err := Write(out, doc); err != nil {
		t.Fatalf("Write: %v", err)
	}
	back, err := Open(out)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer back.Close()
	rd, err := back.Read()
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}

	if got := len(rd.Pages[0].Layers[0].PathObjects); got != 1 {
		t.Fatalf("path objects round-trip = %d, want 1", got)
	}
	if got := len(rd.Pages[1].Layers[0].ImageObjects); got != 1 {
		t.Fatalf("image objects round-trip = %d, want 1", got)
	}
	if rd.Pages[1].Layers[0].ImageObjects[0].ResourceID != "R_0" {
		t.Fatalf("image ResourceID round-trip = %q, want R_0", rd.Pages[1].Layers[0].ImageObjects[0].ResourceID)
	}
}

func TestWrite_PreservesManifestMetadata(t *testing.T) {
	src := openSample(t)
	defer src.Close()
	doc, err := src.Read()
	if err != nil {
		t.Fatalf("read source: %v", err)
	}

	out := t.TempDir() + "/meta.ofd"
	if err := Write(out, doc); err != nil {
		t.Fatalf("Write: %v", err)
	}
	back, err := Open(out)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer back.Close()
	rd, err := back.Read()
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if rd.DocBody.DocInfo.Title != wantSampleTitle {
		t.Fatalf("title round-trip = %q, want %q", rd.DocBody.DocInfo.Title, wantSampleTitle)
	}
	if rd.Version != "1.1" || rd.DocType != "OFD" {
		t.Fatalf("version/docType round-trip = %q/%q, want 1.1/OFD", rd.Version, rd.DocType)
	}
	if rd.DocRoot != "Doc_0/Document.xml" {
		t.Fatalf("DocRoot round-trip = %q, want Doc_0/Document.xml", rd.DocRoot)
	}
}

func TestWrite_NilDocumentErrors(t *testing.T) {
	if err := Write(t.TempDir()+"/nil.ofd", nil); err == nil {
		t.Fatalf("Write(nil) returned nil error, want error")
	}
}

// TestWrite_DefaultsPageIdentity is the regression test for the v0.1.0 defect
// where writeDocumentXML and writePageContent computed two different default
// Content.xml locations for a page with no ID/BaseLoc (manifest said
// Pages/Page_0/Content.xml, the member was written at Pages/Content.xml), so
// the writer's own output could not be read back.
func TestWrite_DefaultsPageIdentity(t *testing.T) {
	doc := &OfdDocument{
		Pages: []Page{{
			Area: &Box{X: 0, Y: 0, W: 210, H: 297},
			Layers: []Layer{{
				ID:   "Layer_0",
				Type: "Body",
				TextObjects: []TextObject{
					{ID: "Text_0", Box: Box{X: 10, Y: 10, W: 50, H: 8}, Text: "hello"},
				},
			}},
		}},
	}
	out := t.TempDir() + "/defaults.ofd"
	if err := Write(out, doc); err != nil {
		t.Fatalf("Write: %v", err)
	}

	back, err := Open(out)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer back.Close()
	rd, err := back.Read()
	if err != nil {
		t.Fatalf("re-read written defaults (v0.1.0 failed here with 'member not found'): %v", err)
	}
	if rd.PageCount() != 1 {
		t.Fatalf("page count = %d, want 1", rd.PageCount())
	}
	if rd.Pages[0].ID != "Page_0" {
		t.Fatalf("defaulted page ID = %q, want Page_0", rd.Pages[0].ID)
	}
	if rd.Pages[0].Area == nil || rd.Pages[0].Area.W != 210 {
		t.Fatalf("page area = %+v, want W=210", rd.Pages[0].Area)
	}
	if got := rd.Pages[0].Layers[0].TextObjects[0].Text; got != "hello" {
		t.Fatalf("text round-trip = %q, want %q", got, "hello")
	}
}

func TestWrite_EmptyDocumentIsReadable(t *testing.T) {
	// A programmatically-constructed empty document still round-trips through
	// the read path, validating the writer's defaulting for missing fields.
	doc := &OfdDocument{}
	out := t.TempDir() + "/empty.ofd"
	if err := Write(out, doc); err != nil {
		t.Fatalf("Write empty: %v", err)
	}
	back, err := Open(out)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer back.Close()
	rd, err := back.Read()
	if err != nil {
		t.Fatalf("re-read empty: %v", err)
	}
	if rd.PageCount() != 0 {
		t.Fatalf("empty doc page count = %d, want 0", rd.PageCount())
	}
}

// TestWritePreserving_InPlaceRejected is the regression test for the v0.2.0
// defect where `ofdcli write -i X.ofd -o X.ofd` truncated the still-open
// source archive with os.Create before copyUnmodeledMembers read from it:
// the command failed with "ofd: open member ...: EOF" and the input was left
// at 0 bytes. The same-file invocation must be refused before any file is
// touched.
func TestWritePreserving_InPlaceRejected(t *testing.T) {
	c := openSample(t)
	defer c.Close()

	doc, err := c.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	p := c.Path()
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read input: %v", err)
	}

	if err := WritePreserving(p, doc, c); err == nil {
		t.Fatalf("WritePreserving onto its own source container returned nil error, want same-file error")
	}

	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("re-read input: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("REPRODUCED (v0.2.0 truncated the input to 0 bytes): input was modified by the rejected write, %d -> %d bytes", len(before), len(after))
	}
}

// resMapFixture is a one-page OFD whose CommonData declares the standard
// resource-map members (PublicRes.xml for fonts, DocumentRes.xml for media)
// and whose ImageObject references a resource declared in DocumentRes.xml.
func resMapFixture() []zipEntry {
	return []zipEntry{
		{name: "OFD.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:OFD xmlns:ofd="http://www.ofdspec.org/2016" DocType="OFD" Version="1.1">
  <ofd:DocBody>
    <ofd:DocInfo><ofd:DocID>resmap-0001</ofd:DocID><ofd:Title>带资源映射的红头文件</ofd:Title></ofd:DocInfo>
    <ofd:DocRoot>Doc_0/Document.xml</ofd:DocRoot>
  </ofd:DocBody>
</ofd:OFD>
`)},
		{name: "Doc_0/Document.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Document xmlns:ofd="http://www.ofdspec.org/2016" ID="Doc_0">
  <ofd:CommonData>
    <ofd:PageArea><ofd:PhysicalBox>0 0 210 297</ofd:PhysicalBox></ofd:PageArea>
    <ofd:PublicRes ID="20" BaseLoc="PublicRes.xml"/>
    <ofd:DocumentRes ID="21" BaseLoc="DocumentRes.xml"/>
  </ofd:CommonData>
  <ofd:Pages><ofd:Page ID="Page_0" BaseLoc="Pages/Page_0/Content.xml"/></ofd:Pages>
</ofd:Document>
`)},
		{name: "Doc_0/Pages/Page_0/Content.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Page xmlns:ofd="http://www.ofdspec.org/2016" ID="Page_0">
  <ofd:Area Boundary="0 0 210 297"/>
  <ofd:Content><ofd:Layer ID="Layer_0" Type="Body">
    <ofd:ImageObject ID="Img_0" Boundary="80 120 50 50" ResourceID="R_0"/>
  </ofd:Layer></ofd:Content>
</ofd:Page>
`)},
		{name: "Doc_0/PublicRes.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:ResBase xmlns:ofd="http://www.ofdspec.org/2016" ID="20">
  <ofd:Fonts><ofd:Font ID="30" FontName="宋体" FamilyName="宋体"/></ofd:Fonts>
</ofd:ResBase>
`)},
		{name: "Doc_0/DocumentRes.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:ResBase xmlns:ofd="http://www.ofdspec.org/2016" ID="21">
  <ofd:MultiMedias><ofd:MultiMedia ID="R_0" Type="Image" Format="PNG"><ofd:MediaFile>Res/R_0/seal.png</ofd:MediaFile></ofd:MultiMedia></ofd:MultiMedias>
</ofd:ResBase>
`)},
		{name: "Doc_0/Res/R_0/seal.png", data: sealBytes},
	}
}

// corruptMemberCRC flips the CRC32 recorded in the central directory for one
// member, so reading that member fails with a checksum error. Any source
// member the writer cannot copy verbatim reaches the same code path.
func corruptMemberCRC(t *testing.T, zipPath, member string) {
	t.Helper()
	data, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", zipPath, err)
	}
	sig := []byte{'P', 'K', 1, 2} // central directory header
	for off := 0; ; {
		i := bytes.Index(data[off:], sig)
		if i < 0 {
			break
		}
		rec := off + i
		if rec+46 > len(data) {
			break
		}
		nameLen := int(binary.LittleEndian.Uint16(data[rec+28:]))
		if rec+46+nameLen <= len(data) && string(data[rec+46:rec+46+nameLen]) == member {
			data[rec+16] ^= 0xFF // four-byte CRC32 field of the central header
			if err := os.WriteFile(zipPath, data, 0o644); err != nil {
				t.Fatalf("rewrite fixture %s: %v", zipPath, err)
			}
			return
		}
		off = rec + 4
	}
	t.Fatalf("member %q not found in the central directory of %s", member, zipPath)
}

// TestWritePreserving_FailedWriteLeavesDestinationIntact is the regression test
// for the v0.3.0 defect where writeArchive called os.Create(outPath) before
// packaging: a failure while copying an unreadable source member had already
// truncated the destination, so the caller's previously valid document at that
// path was destroyed (0 bytes) while the command exited non-zero.
func TestWritePreserving_FailedWriteLeavesDestinationIntact(t *testing.T) {
	srcPath := buildZip(t, resFixture())
	corruptMemberCRC(t, srcPath, "Doc_0/Res/R_0/seal.png")

	src, err := Open(srcPath)
	if err != nil {
		t.Fatalf("Open source: %v", err)
	}
	defer src.Close()
	doc, err := src.Read() // the modelled members are intact
	if err != nil {
		t.Fatalf("Read source: %v", err)
	}

	// A previously written document whose bytes must survive the failed rewrite.
	dir := t.TempDir()
	dst := filepath.Join(dir, "existing.ofd")
	if err := Write(dst, doc); err != nil {
		t.Fatalf("seed destination: %v", err)
	}
	before, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read seeded destination: %v", err)
	}

	if err := WritePreserving(dst, doc, src); err == nil {
		t.Fatalf("WritePreserving with an uncopyable source member returned nil error, want error")
	}

	after, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("re-read destination: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("REPRODUCED (v0.3.0 destroyed the destination): existing document changed %d -> %d bytes after a failed rewrite", len(before), len(after))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("list destination dir: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".ofdcli-") {
			t.Fatalf("failed rewrite left temp file %q behind", e.Name())
		}
	}
}

// TestWritePreserving_KeepsCommonResourceMaps is the regression test for the
// v0.3.0 defect where writeDocumentXML rebuilt CommonData with PageArea alone:
// the PublicRes/DocumentRes declarations were dropped even though the maps
// themselves were copied over verbatim, leaving every ImageObject ResourceID
// unresolvable in the rewritten archive.
func TestWritePreserving_KeepsCommonResourceMaps(t *testing.T) {
	src, err := Open(buildZip(t, resMapFixture()))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer src.Close()
	doc, err := src.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	out := filepath.Join(t.TempDir(), "resmap.ofd")
	if err := WritePreserving(out, doc, src); err != nil {
		t.Fatalf("WritePreserving: %v", err)
	}

	docXML := string(readMemberBytes(t, out, "Doc_0/Document.xml"))
	for _, want := range []string{
		`<PublicRes ID="20" BaseLoc="PublicRes.xml">`,
		`<DocumentRes ID="21" BaseLoc="DocumentRes.xml">`,
	} {
		if !strings.Contains(docXML, want) {
			t.Fatalf("REPRODUCED (v0.3.0 dropped the resource-map declaration): Document.xml lacks %q:\n%s", want, docXML)
		}
	}
	for _, member := range []string{"Doc_0/PublicRes.xml", "Doc_0/DocumentRes.xml"} {
		if got := readMemberBytes(t, out, member); len(got) == 0 {
			t.Fatalf("resource map %q is empty in the rewritten archive", member)
		}
	}
}

// TestWrite_OmitsAbsentResourceMaps guards the other half of the contract: a
// document that declares no resource maps must not gain empty PublicRes /
// DocumentRes elements on write.
func TestWrite_OmitsAbsentResourceMaps(t *testing.T) {
	src := openSample(t)
	defer src.Close()
	doc, err := src.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	out := filepath.Join(t.TempDir(), "nomaps.ofd")
	if err := Write(out, doc); err != nil {
		t.Fatalf("Write: %v", err)
	}
	docXML := string(readMemberBytes(t, out, "Doc_0/Document.xml"))
	if strings.Contains(docXML, "PublicRes") || strings.Contains(docXML, "DocumentRes") {
		t.Fatalf("wrote a resource-map element for a document without one:\n%s", docXML)
	}
}
