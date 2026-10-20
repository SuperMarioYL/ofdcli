package ofd

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"
)

// CommonData tests — the regression suite for the v0.2.0 defect where the
// page-area default was read from and written back as a non-compliant <Common>
// element (GB/T 33190 names it CommonData) with a fabricated
// <PhysicalBox>0 0 0 0</PhysicalBox>, so compliant producers' default page
// area was silently dropped on read and `ofdcli info` printed no page size.

// commonDataFixtureEntries builds a standard-compliant OFD whose Document.xml
// carries <CommonData><PageArea><PhysicalBox>0 0 210 297</PhysicalBox> and
// whose page declares no <Area> of its own (it relies on the default).
func commonDataFixtureEntries() []zipEntry {
	return []zipEntry{
		{name: "OFD.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:OFD xmlns:ofd="http://www.ofdspec.org/2016" DocType="OFD" Version="1.1">
  <ofd:DocBody>
    <ofd:DocInfo><ofd:Title>CommonData默认页面尺寸</ofd:Title></ofd:DocInfo>
    <ofd:DocRoot>Doc_0/Document.xml</ofd:DocRoot>
  </ofd:DocBody>
</ofd:OFD>
`)},
		{name: "Doc_0/Document.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Document xmlns:ofd="http://www.ofdspec.org/2016" ID="Doc_0">
  <ofd:CommonData>
    <ofd:PageArea>
      <ofd:PhysicalBox>0 0 210 297</ofd:PhysicalBox>
    </ofd:PageArea>
  </ofd:CommonData>
  <ofd:Pages>
    <ofd:Page ID="Page_0" BaseLoc="Pages/Page_0/Content.xml"/>
  </ofd:Pages>
</ofd:Document>
`)},
		{name: "Doc_0/Pages/Page_0/Content.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Page xmlns:ofd="http://www.ofdspec.org/2016" ID="Page_0">
  <ofd:Content>
    <ofd:Layer ID="Layer_0" Type="Body">
      <ofd:TextObject ID="Text_0" Boundary="56.7 20 96.6 40" Font="Font_0" Size="22">
        <ofd:TextCode>正文内容</ofd:TextCode>
      </ofd:TextObject>
    </ofd:Layer>
  </ofd:Content>
</ofd:Page>
`)},
	}
}

// readMemberBytes extracts one member's bytes from an OFD zip on disk.
func readMemberBytes(t *testing.T, path, name string) []byte {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer zr.Close()
	for _, f := range zr.Reader.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("open member %q: %v", name, err)
			}
			defer rc.Close()
			b, err := io.ReadAll(rc)
			if err != nil {
				t.Fatalf("read member %q: %v", name, err)
			}
			return b
		}
	}
	t.Fatalf("member %q not found in %s", name, path)
	return nil
}

func TestCommonData_ReadsDefaultPageArea(t *testing.T) {
	c, err := Open(buildZip(t, commonDataFixtureEntries()))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer c.Close()
	doc, err := c.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if doc.Common.PhysicalBox.W != 210 || doc.Common.PhysicalBox.H != 297 {
		t.Fatalf("REPRODUCED (v0.2.0 ignored <CommonData>): Common.PhysicalBox = %+v, want W=210 H=297", doc.Common.PhysicalBox)
	}
}

func TestCommonData_WriteRoundTrip(t *testing.T) {
	c, err := Open(buildZip(t, commonDataFixtureEntries()))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer c.Close()
	doc, err := c.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	out := t.TempDir() + "/commondata.ofd"
	if err := Write(out, doc); err != nil {
		t.Fatalf("Write: %v", err)
	}

	docXML := readMemberBytes(t, out, "Doc_0/Document.xml")
	if !bytes.Contains(docXML, []byte("<CommonData>")) {
		t.Fatalf("written Document.xml lacks the compliant <CommonData> element:\n%s", docXML)
	}
	if bytes.Contains(docXML, []byte("<Common>")) {
		t.Fatalf("written Document.xml still carries the non-compliant <Common> element:\n%s", docXML)
	}
	if bytes.Contains(docXML, []byte("0 0 0 0")) {
		t.Fatalf("written Document.xml fabricates a zero PhysicalBox:\n%s", docXML)
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
	if rd.Common.PhysicalBox.W != 210 || rd.Common.PhysicalBox.H != 297 {
		t.Fatalf("round-tripped Common.PhysicalBox = %+v, want W=210 H=297", rd.Common.PhysicalBox)
	}
}

func TestCommonData_NoBoxOmitsCommonData(t *testing.T) {
	// A model with no page-area default must not gain a fabricated
	// <CommonData><PageArea><PhysicalBox>0 0 0 0</PhysicalBox>, and written
	// page refs must carry no empty <PageArea></PageArea> (encoding/xml
	// ignores omitempty on nested element paths).
	doc := &OfdDocument{
		Pages: []Page{{
			ID: "Page_0",
			Layers: []Layer{{
				ID:          "Layer_0",
				Type:        "Body",
				TextObjects: []TextObject{{ID: "Text_0", Box: Box{X: 10, Y: 10, W: 50, H: 8}, Text: "hello"}},
			}},
		}},
	}
	out := t.TempDir() + "/nobox.ofd"
	if err := Write(out, doc); err != nil {
		t.Fatalf("Write: %v", err)
	}
	docXML := readMemberBytes(t, out, "Doc_0/Document.xml")
	if bytes.Contains(docXML, []byte("<CommonData>")) {
		t.Fatalf("REPRODUCED (v0.2.0 fabricated the element): Document.xml gains <CommonData> for a box-less model:\n%s", docXML)
	}
	if bytes.Contains(docXML, []byte("<PageArea")) {
		t.Fatalf("REPRODUCED (v0.2.0 emitted an empty <PageArea></PageArea> per page):\n%s", docXML)
	}
}
