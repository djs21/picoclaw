# PicoClaw Documentation

PicoClaw documentation is organized by document type. Documentation is written in English only: there is one canonical file per document, with no localized variants.

This file describes the recommended documentation layout and what `make lint-docs` checks locally.

## Reader Navigation

If you are browsing docs rather than reorganizing them, start with these directory indexes:

- [Guides](guides/README.md): setup, configuration, provider, and workflow guides.
- [Reference](reference/README.md): precise configuration and behavior reference.
- [Operations](operations/README.md): debugging and troubleshooting material.
- [Security](security/README.md): security-focused guides and controls.
- [Architecture](architecture/README.md): implementation notes and internal design docs.
- [Migration](migration/README.md): upgrade and migration notes.

For channel-specific setup, start with [Chat Apps Configuration](guides/chat-apps.md) and then drill into `docs/channels/<name>/README.md` as needed.

## Principles

- Documentation is English-only. Do not add translated siblings such as `configuration.zh.md` or `README.pt-br.md`.
- Do not create language buckets such as `docs/zh/`, `docs/fr/`, or `docs/guides/ja/`.
- Choose the document type directory first: `docs/<type>/<name>.md`.
- Keep module-specific docs next to the code they describe instead of moving them into `docs/`.

## Recommended Directories

- `README.md`: English project entry document at the repository root.
- `docs/guides/`: setup and usage guides.
- `docs/reference/`: reference material and detailed configuration docs.
- `docs/operations/`: debugging and troubleshooting docs.
- `docs/security/`: security-related documentation.
- `docs/architecture/`: architecture and internal design notes.
- `docs/channels/`: channel-specific integration guides.
- `docs/design/`: design proposals and investigations.
- `docs/migration/`: migration notes.

## Recommended Naming

- Use the plain English filename with no locale marker:
  - `README.md`
  - `configuration.md`
- Avoid locale markers in any position or case: `README.zh.md`, `configuration_ja.md`, `config.ZH.md`, and `README_pt-br.md` are all rejected.

## Common Patterns To Avoid

- Localized documentation files at any depth under `docs/`
  - Keep a single English document instead.
- Root-level translated entry docs such as `README.zh.md` or `CONTRIBUTING.fr.md`
  - Keep the English `README.md` and `CONTRIBUTING.md` only.
- Language directories under `docs/` such as `docs/zh/`, `docs/ZH/`, or `docs/ja/`
  - Use `docs/<type>/<name>.md` instead.
- Nested locale buckets such as `docs/guides/zh/configuration.md` or `docs/channels/telegram/zh/README.md`
  - Use `docs/<type>/<name>.md` beside the other English docs.
- Legacy translation filenames such as `README_zh.md` or `README_CN.md`
  - Remove the translation and keep the English source.

## Code-Adjacent Docs

Keep documentation next to the implementation when it primarily describes a package, command, example, or subproject.

Examples:

- `pkg/**/README.md`
- `cmd/**/README.md`
- `web/README.md`
- `examples/**/README.md`

These files follow the same English-only rule.

## Adding a New Document

1. Pick the correct document type directory.
2. Create the English source file.
3. Update links from existing docs when the new doc becomes a navigation target.
4. Run `make lint-docs` locally when adding or moving docs.

## Validation

Run:

```bash
make lint-docs
```

The local docs linter currently checks these common cases:

- no locale-marked documentation filenames anywhere in the repository, in any case or with either `.` or `_` separators
- no `docs/<locale>/` language buckets, regardless of case
- no nested locale buckets under typed docs directories
- no extra Markdown files directly under `docs/` except `docs/README.md` and the DOX `docs/AGENTS.md`

`docs/design/` is the only exception: `hook-system-design.zh.md` and `issue-783-investigation-and-fix-plan.zh.md` are Chinese-only documents that never had an English source and await a translate-or-remove decision.

`make lint-docs` is a local consistency check for common naming and placement mistakes. It helps contributors stay close to the recommended layout, but it is not intended to describe every acceptable documentation pattern in the repository.

When a check fails, `make lint-docs` prints the failing path, the reason, and a suggested fix.

If you change these recommendations or want the local linter to reflect them more closely, update this file and `scripts/lint-docs.sh` together.
