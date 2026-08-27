package ofd

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test helpers shared by parser_test.go and writer_test.go.
//
// The sample OFD lives under testdata/sample/ as a real, reviewable member
// tree. buildSampleOFD zips that tree into a temporary .ofd file the same way
// a producer would, so tests exercise the real archive/zip path.

const sampleDir = "testdata/sample"

// buildSampleOFD zips testdata/sample into a temp .ofd file and returns its path.
func buildSampleOFD(t *testing.T) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "sample.ofd")
	if err := zipDir(sampleDir, dst); err != nil {
		t.Fatalf("zip sample: %v", err)
	}
	return dst
}

// writeFile writes data to a path, creating/truncating it.
func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

// zipDir writes every file under srcRoot into a zip at dstPath, preserving
// the relative path within srcRoot as the member name.
func zipDir(srcRoot, dstPath string) error {
	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := zip.NewWriter(out)
	defer zw.Close()
	return filepath.Walk(srcRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(srcRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		w, err := zw.Create(rel)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(w, f)
		return err
	})
}

// wantSampleTitle is the document title carried by the sample OFD.
const wantSampleTitle = "关于召开项目推进会的通知"

// wantPageTexts is the concatenated text block per page index, in order.
var wantPageTexts = [][]string{
	{"关于召开项目推进会的通知", "各有关单位："},
	{"会议时间：2026年9月1日"},
}

// textByPage collects the text of every text object on every page, preserving
// order, for assertion against wantPageTexts.
func textByPage(doc *OfdDocument) [][]string {
	out := make([][]string, len(doc.Pages))
	for i, p := range doc.Pages {
		for _, l := range p.Layers {
			for _, t := range l.TextObjects {
				out[i] = append(out[i], t.Text)
			}
		}
	}
	return out
}

// requireEqualText fails the test if the document's per-page text does not
// match wantPageTexts.
func requireEqualText(t *testing.T, doc *OfdDocument) {
	t.Helper()
	got := textByPage(doc)
	if len(got) != len(wantPageTexts) {
		t.Fatalf("page count = %d, want %d", len(got), len(wantPageTexts))
	}
	for i := range got {
		if strings.Join(got[i], "|") != strings.Join(wantPageTexts[i], "|") {
			t.Fatalf("page %d text = %v, want %v", i, got[i], wantPageTexts[i])
		}
	}
}
