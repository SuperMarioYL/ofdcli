# examples

A committed sample OFD (GB/T 33190) — a two-page 政务 红头文件.

```bash
ofdcli read examples/sample.ofd          # structured JSON
ofdcli info examples/sample.ofd          # document structure
ofdcli write -i examples/sample.ofd -o out.ofd   # normalised round-trip
```
