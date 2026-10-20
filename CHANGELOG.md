# Changelog

All notable changes to this project are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-10-21

### Fixed

- `ofdcli write -i X.ofd -o X.ofd` no longer destroys the input: the writer
  refuses a source and output that resolve to the same file before any file is
  touched. Previously `os.Create` truncated the still-open source archive
  mid-command — the command failed with "ofd: open member ...: EOF" and the
  document was left at 0 bytes.
- The `Document.xml` page-area default is now read from the standard's
  `<CommonData>` element (GB/T 33190 CT_CommonData). Previously the reader
  matched only the non-compliant `<Common>` name ofdcli itself wrote, so
  compliant producers' default page area was silently dropped (`ofdcli info`
  printed no page size), and the writer emitted `<Common>` plus a fabricated
  `<PhysicalBox>0 0 0 0</PhysicalBox>` and an empty `<PageArea></PageArea>`
  on every page. Archives written by older ofdcli versions still read via a
  legacy fallback; the writer now emits only the compliant form, and only
  when the model carries a box.
- The container manifest is resolved case-insensitively on read, so an
  `OFD.XML` archive no longer passes `ofdcli`'s own recognition gate and then
  fails with "member \"OFD.xml\" not found in archive".


## [0.2.0] - 2026-09-15

### Fixed

- `ofdcli write` no longer drops unmodeled archive members: Res/ fonts, images,
  attachments and custom parts are copied from the source document verbatim, so
  a round-tripped document's `ResourceID` references (e.g. a 印章 seal image)
  stay resolvable in reference viewers instead of dangling.
- The writer computes one default `Content.xml` location per page instead of
  two that could disagree: a programmatically authored page without an explicit
  ID/BaseLoc previously produced a manifest pointing at a member the writer
  never wrote, and the output failed `ofdcli read` with "member not found".
- Page geometry declared on the `<Page>` element (`PageArea/PhysicalBox`) in
  `Document.xml` is now read into the model; `ofdcli info` no longer reports
  page size "(unset)" for documents that declare their page box there, and the
  value survives write round-trips.

### Added

- `pkg/ofd.WritePreserving(out, doc, src)` — write API that additionally
  carries unmodeled members from a source container (used by the `write`
  command; plain `Write` keeps its model-only contract).
- This changelog and a version-lockstep test tying the `VERSION` file, the
  `ofdcli --version` output and the changelog's latest section together.

## [0.1.0] - 2026-08-27

### Added

- `ofdcli read` — parse an OFD (GB/T 33190) ZIP container
  (OFD.xml → Doc_N/Document.xml → Pages/Page_N/Content.xml) and emit the
  agent-callable JSON shape `{pages:[{page,blocks:[{text,boundary}]}]}`.
- `ofdcli info` — human-readable document structure and metadata inspection.
- `ofdcli write` — read an OFD and re-emit it as a structurally normalised,
  GB/T 33190-compliant container.

[Unreleased]: https://github.com/SuperMarioYL/ofdcli/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/SuperMarioYL/ofdcli/releases/tag/v0.3.0
[0.2.0]: https://github.com/SuperMarioYL/ofdcli/releases/tag/v0.2.0
[0.1.0]: https://github.com/SuperMarioYL/ofdcli/releases/tag/v0.1.0
