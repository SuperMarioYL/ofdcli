[简体中文](./README.md) · [Website](https://ofdcli.lei6393.com) · [GitHub](https://github.com/SuperMarioYL/ofdcli)

<picture>
  <source media="(max-width: 600px) and (prefers-color-scheme: dark)" srcset="./assets/presentation/hero-mobile-dark.svg">
  <source media="(max-width: 600px)" srcset="./assets/presentation/hero-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="./assets/presentation/hero-dark.svg">
  <img src="./assets/presentation/hero-light.svg" width="960" alt="Hero diagram">
</picture>

# ofdcli

**Read OFD documents from a command line.**

ofdcli opens OFD ZIP/XML containers and exposes text blocks and page boundaries as structured JSON.

## Why use it

Automation needs document text and layout coordinates without opening an office application. A small CLI gives scripts a predictable input path and machine-readable result.

- **JSON for scripts** — Text blocks and coordinates are machine readable.
- **Local document path** — The CLI works directly on a local archive.
- **One supported model** — Read, metadata and rewrite share document structures.

## Architecture

<picture>
  <source media="(max-width: 600px) and (prefers-color-scheme: dark)" srcset="./assets/presentation/architecture-mobile-dark.svg">
  <source media="(max-width: 600px)" srcset="./assets/presentation/architecture-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="./assets/presentation/architecture-dark.svg">
  <img src="./assets/presentation/architecture-light.svg" width="960" alt="Architecture diagram">
</picture>

The reader opens the archive, resolves the OFD manifest and document/page XML chain, then constructs an OfdDocument. read projects text blocks to JSON; info summarizes structure; write serializes the supported model back into an OFD archive.

| Component | Responsibility |
| --- | --- |
| `OFD archive` | pkg/ofd/reader.go |
| `XML parser` | pkg/ofd/parser.go |
| `Document model` | pkg/ofd/document.go |
| `Read / info / write` | cmd/ofdcli |

## Install and quickstart

Build with the version declared in the repository manifest. Run the example from the repository root.

```bash
git clone https://github.com/SuperMarioYL/ofdcli.git
cd ofdcli
go build ./cmd/ofdcli
```

Run read against the complete examples/sample.ofd archive and inspect its two pages of text blocks.

```bash
go run ./cmd/ofdcli read examples/sample.ofd
```

## Recorded demo

<picture>
  <source media="(max-width: 600px) and (prefers-color-scheme: dark)" srcset="./assets/presentation/process-mobile-dark.svg">
  <source media="(max-width: 600px)" srcset="./assets/presentation/process-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="./assets/presentation/process-dark.svg">
  <img src="./assets/presentation/process-light.svg" width="960" alt="Process diagram">
</picture>

The sample document produces two pages of text blocks with millimeter boundaries.

```text
{"pages":[{"page":1,"id":"Page_0","blocks":[{"text":"关于召开项目推进会的通知","boundary":[56.7,20,96.6,40]},{"text":"各有关单位：","boundary":[20,80,170,12]}]},{"page":2,"id":"Page_1","blocks":[{"text":"会议时间：2026年9月1日","boundary":[20,20,170,12]}]}]}
```

The complete command and output are recorded in [docs/demo-results.json](./docs/demo-results.json). Inputs and reproduction code are included in the repository.

## Usage

The CLI exposes the following operations. Commands after the example use your own paths or identifiers.

```bash
go run ./cmd/ofdcli read examples/sample.ofd --pretty
go run ./cmd/ofdcli info examples/sample.ofd
go run ./cmd/ofdcli write -i examples/sample.ofd -o sample.normalized.ofd
```

## Configuration

read takes an OFD path and optional --pretty. info supports --members/-m for archive members. write requires --input/-i and --output/-o; choose a distinct destination to keep the source intact. No daemon or office installation is needed.

## Integrations and responsibilities

<picture>
  <source media="(max-width: 600px) and (prefers-color-scheme: dark)" srcset="./assets/presentation/integrations-mobile-dark.svg">
  <source media="(max-width: 600px)" srcset="./assets/presentation/integrations-mobile-light.svg">
  <source media="(prefers-color-scheme: dark)" srcset="./assets/presentation/integrations-dark.svg">
  <img src="./assets/presentation/integrations-light.svg" width="960" alt="Integrations diagram">
</picture>

The following routes are implemented in the source. Choose the input that matches your task and keep the resulting artifact with your project.

| Route | Implemented role |
| --- | --- |
| OFD ZIP/XML | Container and page tree input |
| Text JSON | Text and millimeter boundaries |
| Info report | Document/page metadata |
| OFD output | Supported model serialization |

## Limits and next steps

- Rewriting preserves the supported model, not every resource or visual detail in arbitrary OFD files. Keep the original and validate rewritten documents with your viewer.
- The sample is a synthetic document; successful parsing is not standards certification or legal validation.
- Template generation, electronic seals and the planned validate command are not implemented here.

Template-based generation and viewer-validated layout fidelity are future work, alongside broader format coverage.

## License and contributions

See [LICENSE](./LICENSE). When reporting an issue, include a minimal input, the command, and the observed output.
