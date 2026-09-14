package ofd

import "testing"

// PageArea tests — the regression suite for the v0.1.0 defect where the
// <Page> element's PageArea/PhysicalBox inside Document.xml was parsed into
// pageRefXML.Area but never projected into the model, so documents declaring
// page geometry there reported page size "(unset)" in `ofdcli info`.

// pageAreaFixtureEntries builds an OFD whose page geometry lives only on the
// <Page> element in Document.xml (Content.xml has no <Area>).
func pageAreaFixtureEntries(withContentArea bool) []zipEntry {
	content := `<?xml version="1.0" encoding="UTF-8"?>
<ofd:Page xmlns:ofd="http://www.ofdspec.org/2016" ID="Page_0">
  <ofd:Content>
    <ofd:Layer ID="Layer_0" Type="Body">
      <ofd:TextObject ID="Text_0" Boundary="56.7 20 96.6 40" Font="Font_0" Size="22">
        <ofd:TextCode X="0" Y="0">正文内容</ofd:TextCode>
      </ofd:TextObject>
    </ofd:Layer>
  </ofd:Content>
</ofd:Page>
`
	if withContentArea {
		content = `<?xml version="1.0" encoding="UTF-8"?>
<ofd:Page xmlns:ofd="http://www.ofdspec.org/2016" ID="Page_0">
  <ofd:Area Boundary="0 0 148 105"/>
  <ofd:Content>
    <ofd:Layer ID="Layer_0" Type="Body">
      <ofd:TextObject ID="Text_0" Boundary="20 20 96.6 40" Font="Font_0" Size="22">
        <ofd:TextCode X="0" Y="0">正文内容</ofd:TextCode>
      </ofd:TextObject>
    </ofd:Layer>
  </ofd:Content>
</ofd:Page>
`
	}
	return []zipEntry{
		{name: "OFD.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:OFD xmlns:ofd="http://www.ofdspec.org/2016" DocType="OFD" Version="1.1">
  <ofd:DocBody>
    <ofd:DocInfo><ofd:DocID>pagearea-0001</ofd:DocID><ofd:Title>页面尺寸在Document.xml的文档</ofd:Title></ofd:DocInfo>
    <ofd:DocRoot>Doc_0/Document.xml</ofd:DocRoot>
  </ofd:DocBody>
</ofd:OFD>
`)},
		{name: "Doc_0/Document.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Document xmlns:ofd="http://www.ofdspec.org/2016" ID="Doc_0">
  <ofd:Pages>
    <ofd:Page ID="Page_0" BaseLoc="Pages/Page_0/Content.xml">
      <ofd:PageArea><ofd:PhysicalBox>0 0 210 297</ofd:PhysicalBox></ofd:PageArea>
    </ofd:Page>
  </ofd:Pages>
</ofd:Document>
`)},
		{name: "Doc_0/Pages/Page_0/Content.xml", data: []byte(content)},
	}
}

func TestRead_PageAreaFromDocumentXML(t *testing.T) {
	c, err := Open(buildZip(t, pageAreaFixtureEntries(false)))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer c.Close()
	doc, err := c.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if doc.PageCount() != 1 {
		t.Fatalf("page count = %d, want 1", doc.PageCount())
	}
	area := doc.Pages[0].Area
	if area == nil {
		t.Fatalf("REPRODUCED (v0.1.0 dropped Document.xml PageArea): page area = nil, want {0 0 210 297}")
	}
	if area.W != 210 || area.H != 297 {
		t.Fatalf("page area = %+v, want W=210 H=297", *area)
	}
}

func TestRead_ContentAreaWinsOverDocumentXML(t *testing.T) {
	// When both locations declare geometry, the page's own <Area> in
	// Content.xml is the more specific one and must win.
	c, err := Open(buildZip(t, pageAreaFixtureEntries(true)))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer c.Close()
	doc, err := c.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	area := doc.Pages[0].Area
	if area == nil || area.W != 148 || area.H != 105 {
		t.Fatalf("page area = %+v, want the Content.xml box {0 0 148 105}", area)
	}
}

func TestReadWrite_PageAreaRoundTrip(t *testing.T) {
	// Geometry declared in Document.xml must survive a write round-trip
	// (the writer serialises p.Area into Content.xml, which is equally
	// valid per GB/T 33190).
	src, err := Open(buildZip(t, pageAreaFixtureEntries(false)))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer src.Close()
	doc, err := src.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	out := t.TempDir() + "/pagearea.ofd"
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
	area := rd.Pages[0].Area
	if area == nil || area.W != 210 || area.H != 297 {
		t.Fatalf("round-tripped page area = %+v, want W=210 H=297", area)
	}
	if got := rd.Pages[0].Layers[0].TextObjects[0].Text; got != "正文内容" {
		t.Fatalf("text round-trip = %q", got)
	}
}
