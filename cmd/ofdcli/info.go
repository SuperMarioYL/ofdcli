package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/SuperMarioYL/ofdcli/pkg/ofd"
)

// infoCmd implements `ofdcli info <file.ofd>`: a human-readable inspection of
// an OFD container — manifest metadata, page count, default page box, archive
// members, and a per-page object breakdown. It complements `read` (the JSON
// agent surface) for a developer eyeballing a document.
var infoCmd = &cobra.Command{
	Use:   "info <file>",
	Short: "查看 OFD 文档结构与元信息",
	Long: `查看 OFD (GB/T 33190) 文档的结构与元信息：版本、标题、作者、页数、
默认页面尺寸、归档成员清单，以及每页的文本/路径/图像对象数量。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := ofd.Open(args[0])
		if err != nil {
			return err
		}
		defer c.Close()

		doc, err := c.Read()
		if err != nil {
			return err
		}

		w := os.Stdout
		abs, _ := filepath.Abs(args[0])
		fmt.Fprintf(w, "file:     %s\n", abs)
		fmt.Fprintf(w, "format:   OFD (GB/T 33190) v%s\n", nonEmpty(doc.Version, "?"))
		fmt.Fprintf(w, "docType:  %s\n", nonEmpty(doc.DocType, "OFD"))
		fmt.Fprintf(w, "docRoot:  %s\n", doc.DocRoot)
		fmt.Fprintf(w, "title:    %s\n", emptyAs(doc.DocBody.DocInfo.Title, "(none)"))
		fmt.Fprintf(w, "author:   %s\n", emptyAs(doc.DocBody.DocInfo.Author, "(none)"))
		fmt.Fprintf(w, "docID:    %s\n", emptyAs(doc.DocBody.DocInfo.DocID, "(none)"))
		fmt.Fprintf(w, "created:  %s\n", emptyAs(doc.DocBody.DocInfo.CreationDate, "(none)"))
		fmt.Fprintf(w, "modified: %s\n", emptyAs(doc.DocBody.DocInfo.ModDate, "(none)"))
		fmt.Fprintf(w, "pages:    %d\n", doc.PageCount())
		if doc.Common.PhysicalBox.W != 0 || doc.Common.PhysicalBox.H != 0 {
			fmt.Fprintf(w, "page size: %s x %s mm (default)\n",
				fmtFloat(doc.Common.PhysicalBox.W), fmtFloat(doc.Common.PhysicalBox.H))
		}
		members := c.MemberNames()
		fmt.Fprintf(w, "members:  %d\n", len(members))

		if doc.PageCount() > 0 {
			fmt.Fprintln(w, "")
			fmt.Fprintln(w, "pages:")
			for i, p := range doc.Pages {
				var textN, pathN, imgN int
				for _, l := range p.Layers {
					textN += len(l.TextObjects)
					pathN += len(l.PathObjects)
					imgN += len(l.ImageObjects)
				}
				size := "(unset)"
				if p.Area != nil {
					size = fmt.Sprintf("%sx%s", fmtFloat(p.Area.W), fmtFloat(p.Area.H))
				}
				fmt.Fprintf(w, "  %d  %-10s  %-12s  texts=%d  paths=%d  images=%d\n",
					i+1, p.ID, size, textN, pathN, imgN)
			}
		}

		if listMembers, _ := cmd.Flags().GetBool("members"); listMembers {
			fmt.Fprintln(w, "")
			fmt.Fprintln(w, "members:")
			for _, m := range members {
				fmt.Fprintf(w, "  %s\n", m)
			}
		}
		return nil
	},
}

func init() {
	infoCmd.Flags().BoolP("members", "m", false, "list all archive members")
}

func nonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func emptyAs(a, b string) string {
	if a == "" {
		return b
	}
	return a
}

func fmtFloat(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%.2f", f)
}
