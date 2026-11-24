package ofd

import (
	"fmt"
	"strconv"
	"strings"
)

// This file holds the small parsers that map a page's XML (Content.xml) onto
// the domain Page/Layer/Object model, plus the shared boundary-box parser.
//
// Boundary attribute format (GB/T 33190): four whitespace-separated numbers
// "x y w h" in millimetres (the OFD default unit). Producers occasionally emit
// multiple spaces; parseBoundary tolerates that.

// parseBox parses an OFD "x y w h" boundary string into a Box.
// An empty string yields the zero Box.
func parseBox(s string) (Box, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Box{}, nil
	}
	fields := strings.Fields(s)
	if len(fields) != 4 {
		return Box{}, fmt.Errorf("ofd: boundary must have 4 numbers, got %d (%q)", len(fields), s)
	}
	var b Box
	var err error
	if b.X, err = strconv.ParseFloat(fields[0], 64); err != nil {
		return Box{}, fmt.Errorf("ofd: bad boundary x %q: %w", fields[0], err)
	}
	if b.Y, err = strconv.ParseFloat(fields[1], 64); err != nil {
		return Box{}, fmt.Errorf("ofd: bad boundary y %q: %w", fields[1], err)
	}
	if b.W, err = strconv.ParseFloat(fields[2], 64); err != nil {
		return Box{}, fmt.Errorf("ofd: bad boundary w %q: %w", fields[2], err)
	}
	if b.H, err = strconv.ParseFloat(fields[3], 64); err != nil {
		return Box{}, fmt.Errorf("ofd: bad boundary h %q: %w", fields[3], err)
	}
	return b, nil
}

// objectBox parses a content object's Boundary attribute. GB/T 33190 requires
// Boundary on CT_Text/CT_Path/CT_Image, so an absent value is an error rather
// than a zero box: reporting [0,0,0,0] would fabricate a measured position and
// size for an object that declared none.
func objectBox(kind, id, boundary string) (Box, error) {
	if strings.TrimSpace(boundary) == "" {
		return Box{}, fmt.Errorf("%s object %q: missing Boundary attribute", kind, id)
	}
	box, err := parseBox(boundary)
	if err != nil {
		return Box{}, fmt.Errorf("%s object %q: %w", kind, id, err)
	}
	return box, nil
}

// buildPage maps a parsed pageXML (Content.xml) onto a domain Page, joining it
// with the page-ref metadata (ID, BaseLoc) carried by Document.xml.
func buildPage(ref pageRefXML, px *pageXML) (Page, error) {
	p := Page{
		ID:      firstNonEmpty(ref.ID, px.ID),
		BaseLoc: ref.BaseLoc,
	}
	if px.Area != nil && strings.TrimSpace(px.Area.Boundary) != "" {
		box, err := parseBox(px.Area.Boundary)
		if err != nil {
			return Page{}, fmt.Errorf("ofd: page %q area: %w", p.ID, err)
		}
		p.Area = &box
	} else if ref.Area != nil && strings.TrimSpace(ref.Area.PhysicalBox) != "" {
		// The producer declared the page geometry on the <Page> element in
		// Document.xml (PageArea/PhysicalBox) and Content.xml carries no usable
		// <Area Boundary> of its own; use that declaration instead of reporting
		// no page size at all.
		box, err := parseBox(ref.Area.PhysicalBox)
		if err != nil {
			return Page{}, fmt.Errorf("ofd: page %q area: %w", p.ID, err)
		}
		p.Area = &box
	}
	if px.Content != nil {
		layers := make([]Layer, 0, len(px.Content.Layers))
		for _, lx := range px.Content.Layers {
			layer, err := buildLayer(lx)
			if err != nil {
				return Page{}, fmt.Errorf("ofd: page %q layer %q: %w", p.ID, lx.ID, err)
			}
			layers = append(layers, layer)
		}
		p.Layers = layers
	}
	return p, nil
}

// buildLayer maps a layerXML onto a domain Layer, projecting each object kind
// onto its domain struct. Text content is the concatenation of every TextCode
// inside a TextObject, in document order.
func buildLayer(lx layerXML) (Layer, error) {
	layer := Layer{ID: lx.ID, Type: lx.Type}

	layer.TextObjects = make([]TextObject, 0, len(lx.TextObjects))
	for _, tx := range lx.TextObjects {
		box, err := objectBox("text", tx.ID, tx.Boundary)
		if err != nil {
			return Layer{}, err
		}
		var b strings.Builder
		for _, tc := range tx.TextCodes {
			b.WriteString(tc.Content)
		}
		layer.TextObjects = append(layer.TextObjects, TextObject{
			ID:   tx.ID,
			Box:  box,
			Font: tx.Font,
			Size: tx.Size,
			Text: b.String(),
		})
	}

	layer.PathObjects = make([]PathObject, 0, len(lx.PathObjects))
	for _, px := range lx.PathObjects {
		box, err := objectBox("path", px.ID, px.Boundary)
		if err != nil {
			return Layer{}, err
		}
		layer.PathObjects = append(layer.PathObjects, PathObject{
			ID:     px.ID,
			Box:    box,
			Points: px.Points,
		})
	}

	layer.ImageObjects = make([]ImageObject, 0, len(lx.ImageObjects))
	for _, ix := range lx.ImageObjects {
		box, err := objectBox("image", ix.ID, ix.Boundary)
		if err != nil {
			return Layer{}, err
		}
		layer.ImageObjects = append(layer.ImageObjects, ImageObject{
			ID:         ix.ID,
			Box:        box,
			ResourceID: ix.ResourceID,
		})
	}

	return layer, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// formatBox is the inverse of parseBox, used by the writer to serialise a Box
// back to the OFD "x y w h" attribute form.
func formatBox(b Box) string {
	return strconv.FormatFloat(b.X, 'f', -1, 64) + " " +
		strconv.FormatFloat(b.Y, 'f', -1, 64) + " " +
		strconv.FormatFloat(b.W, 'f', -1, 64) + " " +
		strconv.FormatFloat(b.H, 'f', -1, 64)
}
