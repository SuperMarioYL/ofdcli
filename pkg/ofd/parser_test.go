package ofd

import (
	"testing"
)

// Parser tests — the m1 read milestone.
//
// These exercise the full read chain: zip container open -> OFD.xml ->
// Doc_0/Document.xml -> Pages/Page_N/Content.xml -> OfdDocument model, and the
// JSON projection the agent tool-call surface consumes.

func TestOpen_ReadsManifestAndDocInfo(t *testing.T) {
	p := buildSampleOFD(t)
	c, err := Open(p)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer c.Close()

	if !c.IsOFD() {
		t.Fatalf("IsOFD = false, want true")
	}
	doc, err := c.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if doc.DocBody.DocInfo.Title != wantSampleTitle {
		t.Fatalf("Title = %q, want %q", doc.DocBody.DocInfo.Title, wantSampleTitle)
	}
	if doc.DocRoot != "Doc_0/Document.xml" {
		t.Fatalf("DocRoot = %q, want Doc_0/Document.xml", doc.DocRoot)
	}
	if doc.Version != "1.1" {
		t.Fatalf("Version = %q, want 1.1", doc.Version)
	}
}

func TestRead_PageTreeAndText(t *testing.T) {
	c := openSample(t)
	defer c.Close()

	doc, err := c.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got := doc.PageCount(); got != 2 {
		t.Fatalf("PageCount = %d, want 2", got)
	}
	if doc.Pages[0].ID != "Page_0" || doc.Pages[1].ID != "Page_1" {
		t.Fatalf("page IDs = %q,%q, want Page_0,Page_1", doc.Pages[0].ID, doc.Pages[1].ID)
	}
	if doc.Pages[0].Area == nil || doc.Pages[0].Area.W != 210 {
		t.Fatalf("page 0 area = %+v, want W=210", doc.Pages[0].Area)
	}
	requireEqualText(t, doc)
}

func TestRead_BoundaryBoxes(t *testing.T) {
	c := openSample(t)
	defer c.Close()

	doc, err := c.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	// First text object on page 0: Boundary="56.7 20 96.6 40".
	first := doc.Pages[0].Layers[0].TextObjects[0]
	if first.Box.X != 56.7 || first.Box.Y != 20 || first.Box.W != 96.6 || first.Box.H != 40 {
		t.Fatalf("text box = %+v, want {56.7 20 96.6 40}", first.Box)
	}
	if first.Size != 22 {
		t.Fatalf("Size = %v, want 22", first.Size)
	}
	if first.Font != "Font_0" {
		t.Fatalf("Font = %q, want Font_0", first.Font)
	}
}

func TestRead_NonTextObjects(t *testing.T) {
	c := openSample(t)
	defer c.Close()

	doc, err := c.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	// Page 0 carries one path object; page 1 carries one image object.
	if n := len(doc.Pages[0].Layers[0].PathObjects); n != 1 {
		t.Fatalf("page 0 path objects = %d, want 1", n)
	}
	if doc.Pages[0].Layers[0].PathObjects[0].ID != "Path_0" {
		t.Fatalf("path ID = %q, want Path_0", doc.Pages[0].Layers[0].PathObjects[0].ID)
	}
	if n := len(doc.Pages[1].Layers[0].ImageObjects); n != 1 {
		t.Fatalf("page 1 image objects = %d, want 1", n)
	}
	if doc.Pages[1].Layers[0].ImageObjects[0].ResourceID != "R_0" {
		t.Fatalf("image ResourceID = %q, want R_0", doc.Pages[1].Layers[0].ImageObjects[0].ResourceID)
	}
}

func TestRead_CommonPhysicalBox(t *testing.T) {
	c := openSample(t)
	defer c.Close()

	doc, err := c.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if doc.Common.PhysicalBox.W != 210 || doc.Common.PhysicalBox.H != 297 {
		t.Fatalf("common PhysicalBox = %+v, want W=210 H=297", doc.Common.PhysicalBox)
	}
}

func TestToJSON_AgentShape(t *testing.T) {
	c := openSample(t)
	defer c.Close()

	doc, err := c.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	j := doc.ToJSON()
	if len(j.Pages) != 2 {
		t.Fatalf("JSON pages = %d, want 2", len(j.Pages))
	}
	if j.Pages[0].Page != 1 || j.Pages[1].Page != 2 {
		t.Fatalf("JSON page numbers = %d,%d, want 1,2", j.Pages[0].Page, j.Pages[1].Page)
	}
	// Page 1 has 2 text blocks; the first carries the title + a 4-number boundary.
	if len(j.Pages[0].Blocks) != 2 {
		t.Fatalf("page 1 blocks = %d, want 2", len(j.Pages[0].Blocks))
	}
	if j.Pages[0].Blocks[0].Text != wantSampleTitle {
		t.Fatalf("block 0 text = %q, want %q", j.Pages[0].Blocks[0].Text, wantSampleTitle)
	}
	if len(j.Pages[0].Blocks[0].Boundary) != 4 || j.Pages[0].Blocks[0].Boundary[0] != 56.7 {
		t.Fatalf("block 0 boundary = %v, want [56.7 20 96.6 40]", j.Pages[0].Blocks[0].Boundary)
	}
}

func TestOpen_MissingOFDXML(t *testing.T) {
	// A zip that is not an OFD container (no OFD.xml member) reads as a manifest error.
	dst := t.TempDir() + "/empty.ofd"
	if err := zipDir("testdata", dst); err != nil { // testdata has no OFD.xml at root
		t.Fatalf("zip: %v", err)
	}
	c, err := Open(dst)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer c.Close()
	if c.IsOFD() {
		t.Fatalf("IsOFD = true on a non-OFD zip, want false")
	}
}

func TestOpen_NotAZip(t *testing.T) {
	tmp := t.TempDir() + "/notazip.ofd"
	if err := writeFile(tmp, []byte("not a zip file")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := Open(tmp); err == nil {
		t.Fatalf("Open on non-zip returned nil error, want error")
	}
}

func openSample(t *testing.T) *Container {
	t.Helper()
	c, err := Open(buildSampleOFD(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return c
}
