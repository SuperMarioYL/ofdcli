package ofd

import (
	"archive/zip"
	"fmt"
	"io"
	"path"
	"strings"
)

// This file is the OFD (GB/T 33190) ZIP-container layer.
//
// OFD is a ZIP package, like OOXML. The container's root member is OFD.xml;
// from there DocBody/DocRoot points at Doc_N/Document.xml, and each page's
// BaseLoc points at Doc_N/Pages/Page_N/Content.xml. Container only owns
// opening/closing the archive and locating members by path; reader.go walks
// the OFD.xml -> Document.xml -> Content.xml chain on top of it.

// Container is an open OFD zip archive.
type Container struct {
	path   string
	reader *zip.ReadCloser
}

// Open opens an OFD file as a zip archive.
func Open(p string) (*Container, error) {
	rc, err := zip.OpenReader(p)
	if err != nil {
		return nil, fmt.Errorf("ofd: open %q: %w", p, err)
	}
	return &Container{path: p, reader: rc}, nil
}

// Close closes the underlying archive reader.
func (c *Container) Close() error {
	if c.reader == nil {
		return nil
	}
	return c.reader.Close()
}

// Reader returns a pointer to the underlying zip.Reader for direct member access.
func (c *Container) Reader() *zip.Reader { return &c.reader.Reader }

// Path returns the on-disk path the container was opened from.
func (c *Container) Path() string { return c.path }

// MemberNames returns every member path in the archive, in archive order.
func (c *Container) MemberNames() []string {
	names := make([]string, 0, len(c.reader.Reader.File))
	for _, f := range c.reader.Reader.File {
		names = append(names, f.Name)
	}
	return names
}

// readMember reads a single member's bytes by exact path.
func (c *Container) readMember(name string) ([]byte, error) {
	for _, f := range c.reader.Reader.File {
		if f.Name == name {
			return readZipFile(f)
		}
	}
	return nil, fmt.Errorf("ofd: member %q not found in archive", name)
}

// readZipFile reads the full contents of one zip file descriptor.
func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("ofd: open member %q: %w", f.Name, err)
	}
	defer rc.Close()
	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("ofd: read member %q: %w", f.Name, err)
	}
	return b, nil
}

// docDir returns the directory portion of the document root path
// (e.g. "Doc_0/Document.xml" -> "Doc_0/"). Page BaseLoc paths are resolved
// relative to this directory.
func docDir(docRoot string) string {
	dir := path.Dir(posixClean(docRoot))
	if dir == "." || dir == "/" {
		return ""
	}
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	return dir
}

// posixClean cleans a path using POSIX semantics (the OFD spec uses forward
// slashes). path.Clean is already POSIX-style on all platforms.
func posixClean(p string) string { return path.Clean(p) }
