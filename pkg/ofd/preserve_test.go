package ofd

import (
	"bytes"
	"testing"
)

// WritePreserving tests — the regression suite for the v0.1.0 defect where
// `ofdcli write` (marketed as 校验/规整/拷贝) dropped every unmodeled archive
// member while keeping the objects that referenced them, so a round-tripped
// 红头文件 lost its 印章 image and the output's ResourceID references dangled.

// sealBytes stands in for a seal image resource carried under Doc_0/Res/.
var sealBytes = []byte("\x89PNG\r\n\x1a\nnot-a-real-png-but-fixed-fixture-bytes")

// resFixture is a one-page OFD whose single image object references the
// resource member Doc_0/Res/R_0/seal.png.
func resFixture() []zipEntry {
	return []zipEntry{
		{name: "OFD.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:OFD xmlns:ofd="http://www.ofdspec.org/2016" DocType="OFD" Version="1.1">
  <ofd:DocBody>
    <ofd:DocInfo><ofd:DocID>res-fix-0001</ofd:DocID><ofd:Title>带公章的红头文件</ofd:Title></ofd:DocInfo>
    <ofd:DocRoot>Doc_0/Document.xml</ofd:DocRoot>
  </ofd:DocBody>
</ofd:OFD>
`)},
		{name: "Doc_0/Document.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Document xmlns:ofd="http://www.ofdspec.org/2016" ID="Doc_0">
  <ofd:Common><ofd:PageArea><ofd:PhysicalBox>0 0 210 297</ofd:PhysicalBox></ofd:PageArea></ofd:Common>
  <ofd:Pages><ofd:Page ID="Page_0" BaseLoc="Pages/Page_0/Content.xml"/></ofd:Pages>
</ofd:Document>
`)},
		{name: "Doc_0/Pages/Page_0/Content.xml", data: []byte(`<?xml version="1.0" encoding="UTF-8"?>
<ofd:Page xmlns:ofd="http://www.ofdspec.org/2016" ID="Page_0">
  <ofd:Area Boundary="0 0 210 297"/>
  <ofd:Content>
    <ofd:Layer ID="Layer_0" Type="Body">
      <ofd:TextObject ID="Text_0" Boundary="56.7 20 96.6 40" Font="Font_0" Size="22">
        <ofd:TextCode X="0" Y="0">关于召开项目推进会的通知</ofd:TextCode>
      </ofd:TextObject>
      <ofd:ImageObject ID="Img_0" Boundary="80 120 50 50" ResourceID="R_0"/>
    </ofd:Layer>
  </ofd:Content>
</ofd:Page>
`)},
		{name: "Doc_0/Res/R_0/seal.png", data: sealBytes},
	}
}

func TestWritePreserving_KeepsUnmodeledMembers(t *testing.T) {
	src, err := Open(buildZip(t, resFixture()))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer src.Close()
	doc, err := src.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	out := t.TempDir() + "/preserved.ofd"
	if err := WritePreserving(out, doc, src); err != nil {
		t.Fatalf("WritePreserving: %v", err)
	}

	back, err := Open(out)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer back.Close()

	// The seal resource survives byte-identical, so the ImageObject's
	// ResourceID="R_0" reference stays resolvable.
	got, err := back.readMember("Doc_0/Res/R_0/seal.png")
	if err != nil {
		t.Fatalf("REPRODUCED (v0.1.0 dropped the Res member): %v", err)
	}
	if !bytes.Equal(got, sealBytes) {
		t.Fatalf("Res member content changed: %d bytes, want %d", len(got), len(sealBytes))
	}

	rd, err := back.Read()
	if err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if got := len(rd.Pages[0].Layers[0].ImageObjects); got != 1 {
		t.Fatalf("image objects = %d, want 1", got)
	}
	if rd.Pages[0].Layers[0].ImageObjects[0].ResourceID != "R_0" {
		t.Fatalf("image ResourceID = %q, want R_0", rd.Pages[0].Layers[0].ImageObjects[0].ResourceID)
	}
	if rd.DocBody.DocInfo.Title != "带公章的红头文件" {
		t.Fatalf("title = %q", rd.DocBody.DocInfo.Title)
	}
}

func TestWritePreserving_DoesNotDuplicateModeledMembers(t *testing.T) {
	src, err := Open(buildZip(t, resFixture()))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer src.Close()
	doc, err := src.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	out := t.TempDir() + "/norepeat.ofd"
	if err := WritePreserving(out, doc, src); err != nil {
		t.Fatalf("WritePreserving: %v", err)
	}
	back, err := Open(out)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer back.Close()

	counts := map[string]int{}
	for _, n := range back.MemberNames() {
		counts[n]++
	}
	for _, modeled := range []string{"OFD.xml", "Doc_0/Document.xml", "Doc_0/Pages/Page_0/Content.xml"} {
		if counts[modeled] != 1 {
			t.Fatalf("modeled member %q appears %d times, want exactly 1", modeled, counts[modeled])
		}
	}
	if len(counts) != 4 {
		t.Fatalf("output members = %v, want exactly the 3 modeled + 1 Res member", back.MemberNames())
	}
}

func TestWrite_ModelOnlyBehaviorUnchanged(t *testing.T) {
	// Plain Write stays model-only: the additive WritePreserving API did not
	// change the public Write contract for pure-model documents.
	src, err := Open(buildZip(t, resFixture()))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer src.Close()
	doc, err := src.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	out := t.TempDir() + "/modelonly.ofd"
	if err := Write(out, doc); err != nil {
		t.Fatalf("Write: %v", err)
	}
	back, err := Open(out)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer back.Close()
	for _, n := range back.MemberNames() {
		if n == "Doc_0/Res/R_0/seal.png" {
			t.Fatalf("plain Write copied the Res member; Write is model-only by contract")
		}
	}

	// WritePreserving with a nil source is rejected rather than silently
	// degrading to Write.
	if err := WritePreserving(out, doc, nil); err == nil {
		t.Fatalf("WritePreserving(nil src) returned nil error, want error")
	}
}
