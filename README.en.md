**English** | [简体中文](./README.md)

<div align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="./assets/hero-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="./assets/hero-light.svg">
    <img src="./assets/hero-light.svg" width="880" alt="Ofdcli — OFD fixed-layout read/write CLI for 信创 agents">
  </picture>
</div>

<p align="center"><sub>The OFD (GB/T 33190) fixed-layout document read/write CLI for 信创 agents — single binary, zero dependencies.</sub></p>

<p align="center">
  <a href="./LICENSE"><img src="https://img.shields.io/github/license/SuperMarioYL/ofdcli?color=blue" alt="license"></a>
  <img src="https://img.shields.io/github/v/release/SuperMarioYL/ofdcli" alt="release">
  <img src="https://img.shields.io/github/actions/workflow/status/SuperMarioYL/ofdcli/ci.yml?label=ci" alt="ci">
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white" alt="go">
  <img src="https://img.shields.io/badge/OFD-GB%2FT%2033190-5E5CE6" alt="ofd">
</p>

> Recommended GitHub topics: `gh repo edit --add-topic ofd --add-topic xinchuang --add-topic gov-doc --add-topic agent-cli --add-topic go`

**信创 automation agents get stuck on OFD — ofdcli lets them read and write GB/T 33190 政务 fixed-layout documents the same way they read and write Word: a single binary, no Office dependency, callable directly from an agent loop.**

<h2><img src="https://api.iconify.design/tabler:topology-star-3.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Architecture</h2>

OFD is a ZIP container (isomorphic to OOXML) whose members are XML: `OFD.xml → Doc_0/Document.xml → Pages/Page_N/Content.xml`. ofdcli maps that tree to agent-callable JSON I/O — one process, no servers, no Kubernetes.

<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./assets/atlas-dark.svg">
  <source media="(prefers-color-scheme: light)" srcset="./assets/atlas-light.svg">
  <img src="./assets/atlas-light.svg" width="880" alt="architecture: OFD container → ofdcli parse/write → JSON/.ofd agent I/O">
</picture>

<h2><img src="https://api.iconify.design/tabler:rocket.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Install</h2>

```bash
go install github.com/SuperMarioYL/ofdcli@latest
```

Or download a single binary from the Release for your platform (Linux/darwin/arm64 — 信创 desktops). No Office/WPS install, no Python runtime.

<h2><img src="https://api.iconify.design/tabler:bolt.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Quickstart</h2>

```bash
git clone https://github.com/SuperMarioYL/ofdcli && cd ofdcli
go build -o ofdcli ./cmd/ofdcli
./ofdcli read examples/sample.ofd          # structured JSON to stdout
```

<details><summary>Sample output</summary>

```json
{"pages":[{"page":1,"id":"Page_0","blocks":[{"text":"关于召开项目推进会的通知","boundary":[56.7,20,96.6,40]},{"text":"各有关单位：","boundary":[20,80,170,12]}]},{"page":2,"id":"Page_1","blocks":[{"text":"会议时间：2026年9月1日","boundary":[20,20,170,12]}]}]}
```
</details>

<h2><img src="https://api.iconify.design/tabler:terminal-2.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Usage</h2>

```bash
# 1. Read an OFD: agent-consumable structured JSON (page -> text block + boundary box, in mm)
ofdcli read red-header.ofd
ofdcli read red-header.ofd --pretty        # indented, readable

# 2. Inspect document structure + metadata (version/title/author/pages/page size/per-page objects)
ofdcli info red-header.ofd
ofdcli info -m red-header.ofd              # also list archive members

# 3. Read-in / write-out normalisation: round-trip the model into a compliant GB/T 33190 container (copy/normalise/verify)
ofdcli write -i red-header.ofd -o red-header.norm.ofd
```

Every subcommand is callable from an agent tool-call loop: `read` emits JSON to stdout, `write` reads `-i` and writes `-o`, `info` gives an agent a structural overview. See `ofdcli --help` for the full command list.

<h2><img src="https://api.iconify.design/tabler:file-zip.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> How it works</h2>

OFD (Open Fixed-layout Document, GB/T 33190) is the mandated fixed-layout archival/exchange format for Chinese 政务 documents — red-header documents (红头文件), contracts, invoices. It is fundamentally **ZIP + XML**, structured as:

```
red-header.ofd                        # a zip
├── OFD.xml                           # container manifest: DocBody -> DocRoot points at Document.xml
└── Doc_0/
    ├── Document.xml                  # document structure: Common + Pages list
    └── Pages/
        ├── Page_0/Content.xml        # page content: Area + Content -> Layer -> objects
        └── Page_1/Content.xml
```

The page tree inside `Content.xml` is `Layer -> TextObject / PathObject / ImageObject`. A `TextObject` carries a `Boundary` ("x y w h", mm) and `TextCode` (the glyph string); ofdcli projects it into agent-callable JSON `{pages:[{page, blocks:[{text, boundary}]}]}`.

This is isomorphic to the global OfficeCLI (29k★, a single binary that reads/writes Word/Excel/PPT) — but OfficeCLI does not cover Chinese 政务 formats. OFD is a separate national standard, unrelated to OOXML. ofdcli fills exactly that China surface: the same agent-CLI shape, for the fixed-layout format 政务 workflows actually mandate.

<h2><img src="https://api.iconify.design/tabler:photo.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Demo</h2>

![demo](assets/demo.gif)

> The demo is driven by `docs/demo.tape` (a vhs script) and rendered to `assets/demo.gif` by `.github/workflows/demo.yml`. The first gif is generated after the first tag / a manual workflow run.

<h2><img src="https://api.iconify.design/tabler:building.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Enterprise</h2>

The OSS core (read / write / info, a single template, self-hosted) is free forever — 信创 means data stays on-prem; the binary is the deployment unit.

The **Enterprise tier** (per-team license, on-prem license-gated) targets system integrators and 政企 IT automation teams, adding:

- Batch OFD generation (red-header / contract / invoice; the OSS core ships one seed template only)
- Template packs (red-header / contract / invoice layout templates)
- Compliance audit logging (every read/write is traced, for 政务 audit)
- Priority format-fix support

Indicative pricing: per-team license ¥3,000–8,000/year (5 seats); project PoC / integration license ¥10,000–50,000. Billed via corporate invoice / license-key, not SaaS. Distributed and listed through Alibaba Cloud / Huawei Cloud marketplaces (domestic clouds). To evaluate, file an issue or reach out via corporate contact.

<h2><img src="https://api.iconify.design/tabler:map-2.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> Roadmap</h2>

- [x] **m1 — Read OFD**: parse the GB/T 33190 ZIP container, extract structured text + page layout from `Doc_0/Document.xml`, emit JSON
- [x] **m1 — Write OFD**: model → compliant XML → normalised zip (read-in round-trip, unit-tested)
- [ ] **m2 — Template generation**: `ofdcli write --template contract --data contract.json -o contract.ofd`, compliance-verified in a reference viewer (数科/福昕)
- [ ] **m3 — Agent CLI**: cobra `read`/`write`/`validate` subcommands + JSON stdin/stdout tool-call surface + asciinema demo + Gitee mirror
- [ ] **v0.2 — WPS native formats** (`.et`/`.wps`/`.dps`) read & write (after de-risking with a real sample)
- [ ] **v0.3 — Electronic seal / 印章** cryptographic embedding

The current release is **m1**: read and read-in round-trip write are implemented and tested; template generation (m2) and the validate subcommand (m3) are on the roadmap.

<h2><img src="https://api.iconify.design/tabler:license.svg?color=%230071E3&width=24" height="22" align="absmiddle" alt=""> License</h2>

MIT — see [LICENSE](./LICENSE). Issues and PRs welcome.

<p align="center"><sub><a href="./LICENSE">MIT</a> © 2026 SuperMarioYL</sub></p>
