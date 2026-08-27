Output assets/demo.gif
Set FontSize 14
Set Width 1200
Set Height 620
Set Padding 20
Set Theme "Dracula"

# ofdcli — 10-minute happy path demo
# Build the single binary, read a 政务 红头文件, inspect it, write a normalised copy.

Type "go build -o ofdcli ./cmd/ofdcli"
Enter
Sleep 2s

Type "./ofdcli --version"
Enter
Sleep 1s

Type "./ofdcli read examples/sample.ofd"
Enter
Sleep 1s

Type "./ofdcli info examples/sample.ofd"
Enter
Sleep 1s

Type "./ofdcli write -i examples/sample.ofd -o /tmp/合同.ofd"
Enter
Sleep 1s

Type "./ofdcli read /tmp/合同.ofd"
Enter
Sleep 1s
