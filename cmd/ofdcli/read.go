package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/SuperMarioYL/ofdcli/pkg/ofd"
)

// readCmd implements `ofdcli read <file.ofd>`: it opens an OFD container,
// walks the GB/T 33190 member chain into a document model, and prints the
// agent-callable JSON shape {pages:[{page,blocks:[{text,boundary}]}]}.
var readCmd = &cobra.Command{
	Use:   "read <file>",
	Short: "读取 OFD，输出结构化 JSON",
	Long: `读取 OFD (GB/T 33190) 文档，按 页 -> 文本块 输出 JSON 到 stdout。

输出形如：
  {"pages":[{"page":1,"id":"Page_0","blocks":[{"text":"...","boundary":[x,y,w,h]}]}]}

边界坐标单位为毫米（OFD 默认单位）。`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		pretty, _ := cmd.Flags().GetBool("pretty")

		c, err := ofd.Open(args[0])
		if err != nil {
			return err
		}
		defer c.Close()

		doc, err := c.Read()
		if err != nil {
			return err
		}

		out := doc.ToJSON()
		var b []byte
		if pretty {
			b, err = json.MarshalIndent(out, "", "  ")
		} else {
			b, err = json.Marshal(out)
		}
		if err != nil {
			return fmt.Errorf("marshal json: %w", err)
		}
		fmt.Fprintln(os.Stdout, string(b))
		return nil
	},
}

func init() {
	readCmd.Flags().BoolP("pretty", "p", false, "pretty-print JSON output")
}
