package ofd

// This file is the agent-callable domain model of an open OFD document tree.
//
// reader.go maps the GB/T 33190 XML structs (xml_types.go) into this model;
// writer.go maps this model back into XML and re-packages it as a compliant
// OFD zip. The model is what the CLI surfaces as JSON to agent tool-calls, so
// it is shaped for stable JSON I/O rather than for pixel-perfect layout
// fidelity (out of scope for v0.1 per the plan).

// OfdDocument is the open OFD container's logical view.
type OfdDocument struct {
	Version string   // OFD spec version, e.g. "1.1"
	DocType string   // e.g. "OFD"
	DocBody DocBody  // first document body (OFD.xml/DocBody[0])
	Common  Common   // shared page-area defaults
	Pages   []Page   // ordered pages from Document.xml/Pages

	// DocRoot is the Document.xml path inside the archive, e.g. "Doc_0/Document.xml".
	// It anchors page BaseLoc paths (which are relative to the Doc directory).
	DocRoot string
}

// DocBody holds the document body pointer and descriptive metadata.
type DocBody struct {
	DocRoot         string  // path to Document.xml
	AnnotationsPage string
	DocInfo         DocInfo
}

// DocInfo is CT_DocInfo projected onto the fields an agent loop reads.
type DocInfo struct {
	DocID        string
	Title        string
	Author       string
	Creator      string
	CreationDate string
	ModDate      string
}

// Common is the document's shared CT_PageArea defaults.
type Common struct {
	PhysicalBox Box // default page physical box (mm)
}

// Box is an "x y w h" boundary box in millimetres (the OFD default unit).
type Box struct {
	X float64
	Y float64
	W float64
	H float64
}

// Page is one page in the document tree.
type Page struct {
	ID      string // page ID (e.g. "Page_0")
	BaseLoc string // content path relative to the Doc directory (e.g. "Pages/Page_0/Content.xml")
	Area    *Box   // page boundary; nil when the producer omitted <Area>
	Layers  []Layer
}

// Layer is one content layer of a page.
type Layer struct {
	ID           string
	Type         string
	TextObjects  []TextObject
	PathObjects  []PathObject
	ImageObjects []ImageObject
}

// TextObject is a run of text: its boundary plus the concatenated glyph string.
type TextObject struct {
	ID       string
	Box      Box    // text block boundary
	Font     string // resource ID of the font (Font_0); resolution to font metadata is out of scope for v0.1
	Size     float64
	Text     string
}

// PathObject is a vector path. Points is the raw point data for v0.1.
type PathObject struct {
	ID     string
	Box    Box
	Points string
}

// ImageObject is an image reference.
type ImageObject struct {
	ID         string
	Box       Box
	ResourceID string
}

// --- JSON output model (agent tool-call surface) ---

// JSONDocument is the JSON shape emitted by `ofdcli read`.
//
//	{"pages":[{"page":1,"blocks":[{"text":"...","boundary":[x,y,w,h]}]}]}
type JSONDocument struct {
	Pages []JSONPage `json:"pages"`
}

// JSONPage is one page in the read output.
type JSONPage struct {
	Page   int         `json:"page"`
	ID     string      `json:"id,omitempty"`
	Blocks []JSONBlock `json:"blocks"`
}

// JSONBlock is one text block on a page.
type JSONBlock struct {
	Text     string    `json:"text"`
	Boundary []float64 `json:"boundary"`
}

// ToJSON projects the document model into the agent-callable JSON shape:
// pages in order, each text object becoming a block with its text + boundary.
func (d *OfdDocument) ToJSON() JSONDocument {
	out := JSONDocument{Pages: make([]JSONPage, 0, len(d.Pages))}
	for i, p := range d.Pages {
		jp := JSONPage{Page: i + 1, ID: p.ID}
		for _, layer := range p.Layers {
			for _, t := range layer.TextObjects {
				jp.Blocks = append(jp.Blocks, JSONBlock{
					Text:     t.Text,
					Boundary: []float64{t.Box.X, t.Box.Y, t.Box.W, t.Box.H},
				})
			}
		}
		out.Pages = append(out.Pages, jp)
	}
	return out
}

// PageCount returns the number of pages in the document.
func (d *OfdDocument) PageCount() int { return len(d.Pages) }
