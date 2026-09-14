package ofd

import (
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
