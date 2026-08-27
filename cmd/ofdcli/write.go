package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/SuperMarioYL/ofdcli/pkg/ofd"
)

// writeCmd implements `ofdcli write -i <in.ofd> -o <out.ofd>`.
//
// m1 scope: read an OFD document and re-emit it through the writer, producing
// a GB/T 33190-compliant, structurally-normalised container that round-trips
// through `ofdcli read`. Template-driven generation from arbitrary JSON input
// (--template / --data) is the m2 milestone and is intentionally not wired here.
var writeCmd = &cobra.Command{
	Use:   "write -i <input.ofd> -o <output.ofd>",
	Short: "写出 OFD 版式文档（m1: 读入-回写规范化）",
	Long: `写出 OFD (GB/T 33190) 版式文档。

m1：读入一个 OFD，经模型规范化后回写为合规容器（可用于校验/规整/拷贝）。
  ofdcli write -i 红头文件.ofd -o 红头文件.norm.ofd

m2（路线图）：基于模板从结构化 JSON 生成新版式文档。
  ofdcli write --template 合同 --data contract.json -o 合同.ofd   # 待实现`,
	RunE: func(cmd *cobra.Command, args []string) error {
		in, _ := cmd.Flags().GetString("input")
		out, _ := cmd.Flags().GetString("output")
		if in == "" || out == "" {
			return fmt.Errorf("both --input (-i) and --output (-o) are required")
		}

		c, err := ofd.Open(in)
		if err != nil {
			return err
		}
		defer c.Close()

		doc, err := c.Read()
		if err != nil {
			return err
		}
		if err := ofd.Write(out, doc); err != nil {
			return err
		}
		fmt.Printf("OK %s (GB/T 33190, %d pages)\n", out, doc.PageCount())
		return nil
	},
}

func init() {
	writeCmd.Flags().StringP("input", "i", "", "input OFD file to read and normalise")
	writeCmd.Flags().StringP("output", "o", "", "output OFD file to write")
}
