package ofd

import "encoding/xml"

// This file declares the XML serialization layer for the OFD (GB/T 33190)
// fixed-layout container.
//
// OFD is a ZIP archive whose members are XML in the OFD namespace
// "http://www.ofdspec.org/2016". The container root manifest OFD.xml points
// (via DocBody/DocRoot) at a per-document Document.xml, which lists Pages;
// each Page's BaseLoc points at a Content.xml holding the page tree
// (Layer -> TextObject/PathObject/ImageObject).
//
// The root element declares the namespace explicitly so the same tag is used on
// read and write. Child fields are tagged by local name only, which makes the
// parser tolerant of OFD producers that vary the namespace prefix while keeping
// the GB/T 33190 namespace URI — local-name matching is stable across producers.

const NameSpace = "http://www.ofdspec.org/2016"

// ofdXML is OFD.xml — the container manifest at the archive root.
type ofdXML struct {
	XMLName xml.Name `xml:"http://www.ofdspec.org/2016 OFD"`
	DocType string   `xml:"DocType,attr,omitempty"`
	Version string    `xml:"Version,attr,omitempty"`
	DocBody []docBodyXML `xml:"DocBody"`
}

// docBodyXML is one CT_DocBody: document metadata + a pointer to the document root.
type docBodyXML struct {
	DocInfo  *docInfoXML `xml:"DocInfo"`
	DocRoot  string      `xml:"DocRoot"`
	AnnotationsPage string `xml:"AnnotationsPage,omitempty"`
}

// docInfoXML is CT_DocInfo — the document's descriptive metadata.
type docInfoXML struct {
	DocID        string `xml:"DocID"`
	Title        string `xml:"Title"`
	Author       string `xml:"Author"`
	Creator      string `xml:"Creator"`
	CreationDate string `xml:"CreationDate"`
	ModDate      string `xml:"ModDate"`
}

// documentXML is Doc_N/Document.xml — the per-document structural root.
type documentXML struct {
	XMLName xml.Name `xml:"http://www.ofdspec.org/2016 Document"`
	ID      string   `xml:"ID,attr,omitempty"`
	Common  *commonXML `xml:"Common"`
	Pages   []pageRefXML `xml:"Pages>Page"`
}

// commonXML holds page-area defaults shared across the document's pages.
type commonXML struct {
	PageArea *pageAreaXML `xml:"PageArea"`
}

// pageAreaXML is CT_PageArea; PhysicalBox is "x y w h" in mm (the OFD default unit).
type pageAreaXML struct {
	PhysicalBox string `xml:"PhysicalBox"`
	ApplicationBox string `xml:"ApplicationBox,omitempty"`
	Application string `xml:"Application,omitempty"`
}

// pageRefXML is a <Page> entry inside Document.xml/Pages. BaseLoc is the
// content path relative to the enclosing Doc_N/ directory.
type pageRefXML struct {
	ID      string `xml:"ID,attr,omitempty"`
	BaseLoc string `xml:"BaseLoc,attr,omitempty"`
	Area    string `xml:"PageArea>PhysicalBox,omitempty"`
}

// pageXML is a page's Content.xml: the page area + a content tree of layers.
type pageXML struct {
	XMLName xml.Name `xml:"http://www.ofdspec.org/2016 Page"`
	ID      string   `xml:"ID,attr,omitempty"`
	Area    *areaXML `xml:"Area"`
	Content *contentXML `xml:"Content"`
}

// areaXML is the page's boundary box ("x y w h" in mm).
type areaXML struct {
	Boundary string `xml:"Boundary,attr,omitempty"`
}

// contentXML holds the ordered layer list of a page.
type contentXML struct {
	Layers []layerXML `xml:"Layer"`
}

// layerXML is CT_Layer. Objects are modeled in three ordered slices matching
// the GB/T 33190 content model; out-of-scope object kinds (e.g. CompositeObject)
// are intentionally not captured for v0.1.
type layerXML struct {
	ID           string          `xml:"ID,attr,omitempty"`
	Type         string          `xml:"Type,attr,omitempty"`
	TextObjects  []textObjectXML `xml:"TextObject"`
	PathObjects  []pathObjectXML `xml:"PathObject"`
	ImageObjects []imageObjectXML `xml:"ImageObject"`
}

// textObjectXML is CT_Text. Boundary is "x y w h"; the visible text is the
// concatenation of all TextCode character data inside the object.
type textObjectXML struct {
	ID       string         `xml:"ID,attr,omitempty"`
	Boundary string         `xml:"Boundary,attr,omitempty"`
	Font     string         `xml:"Font,attr,omitempty"`
	Size     float64        `xml:"Size,attr,omitempty"`
	TextCodes []textCodeXML `xml:"TextCode"`
}

// textCodeXML is CT_TextCode. DeltaX/DeltaY are per-glyph offsets; the element
// character data carries the glyph string itself.
type textCodeXML struct {
	X      string `xml:"X,attr,omitempty"`
	Y      string `xml:"Y,attr,omitempty"`
	Content string `xml:",chardata"`
}

// pathObjectXML is CT_Path. The point data is left raw for v0.1.
type pathObjectXML struct {
	ID       string `xml:"ID,attr,omitempty"`
	Boundary string `xml:"Boundary,attr,omitempty"`
	Points   string `xml:"AbbreviatedLineOrPoint,omitempty"`
}

// imageObjectXML is CT_Image. ResourceID references a resource in the page's
// or document's Res map.
type imageObjectXML struct {
	ID         string `xml:"ID,attr,omitempty"`
	Boundary   string `xml:"Boundary,attr,omitempty"`
	ResourceID string `xml:"ResourceID,attr,omitempty"`
}
