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

// buildPage maps a parsed pageXML (Content.xml) onto a domain Page, joining it
// with the page-ref metadata (ID, BaseLoc) carried by Document.xml.
func buildPage(ref pageRefXML, px *pageXML) (Page, error) {
	p := Page{
		ID:      firstNonEmpty(ref.ID, px.ID),
		BaseLoc: ref.BaseLoc,
	}
	if px.Area != nil {
		box, err := parseBox(px.Area.Boundary)
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
		box, err := parseBox(tx.Boundary)
		if err != nil {
			return Layer{}, fmt.Errorf("text object %q: %w", tx.ID, err)
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
		box, err := parseBox(px.Boundary)
		if err != nil {
			return Layer{}, fmt.Errorf("path object %q: %w", px.ID, err)
		}
		layer.PathObjects = append(layer.PathObjects, PathObject{
			ID:     px.ID,
			Box:    box,
			Points: px.Points,
		})
	}

	layer.ImageObjects = make([]ImageObject, 0, len(lx.ImageObjects))
	for _, ix := range lx.ImageObjects {
		box, err := parseBox(ix.Boundary)
		if err != nil {
			return Layer{}, fmt.Errorf("image object %q: %w", ix.ID, err)
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
