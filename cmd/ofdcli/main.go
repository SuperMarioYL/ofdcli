// Command ofdcli is the 信创 agent CLI for reading and writing OFD
// (GB/T 33190) fixed-layout 政务 documents: a single Go binary with no Office
// dependency, callable from an agent tool-call loop.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is the single source for `ofdcli --version`. The release tag is
// derived from the repo-root VERSION file by publish tooling; this constant
// keeps the binary self-describing between releases.
const version = "0.1.0"

const longDesc = `ofdcli — 信创 agent 的 OFD (GB/T 33190) 政务版式文档读写 CLI

让自动化 agent 像 OfficeCLI 读写 Word/Excel 那样，读写 GB/T 33190 政务版式文档
（红头文件 / 合同 / 发票）：单二进制、无 Office 依赖、可被 agent 直接调用。

OFD 是一个 ZIP 容器（与 OOXML 同构），成员为 XML（OFD.xml -> Doc_0/Document.xml ->
Pages/Page_N/Content.xml）。ofdcli 把这棵树映射为 agent 可调用的 JSON I/O。

子命令：
  read    读取 OFD，输出结构化 JSON（agent tool-call surface）
  write   写出 OFD 版式文档（m1: 读入-回写规范化）
  info    查看 OFD 文档结构与元信息`

var rootCmd = &cobra.Command{
	Use:     "ofdcli",
	Short:   "信创 agent 的 OFD (GB/T 33190) 政务版式文档读写 CLI，单二进制无依赖。",
	Long:    longDesc,
	Version: version,
	// No default Run: print help when invoked with no subcommand.
}

func main() {
	rootCmd.AddCommand(readCmd, writeCmd, infoCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "ofdcli:", err)
		os.Exit(1)
	}
}
