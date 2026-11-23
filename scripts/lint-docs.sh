#!/usr/bin/env bash

# Enforces the documentation layout rules for this repository.
#
# Documentation is English-only. Localized variants of docs are not kept:
# there is one canonical English file per document.

set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

failures=0

error() {
	local path="$1"
	local reason="$2"
	local suggestion="${3:-}"

	echo "docs lint: $path" >&2
	echo "  reason: $reason" >&2
	if [[ -n "$suggestion" ]]; then
		echo "  fix: $suggestion" >&2
	fi
	failures=1
}

# Locale codes that must never be used to name or bucket documentation.
# Keep this list broad enough to catch forgotten translations. Do not replace it
# with a generic two-letter match: channel directories such as docs/channels/qq/
# and docs/channels/vk/ are legitimate and would be false positives.
locale_codes='en|it|zh|fr|ja|ms|pt|pt-br|vi|es|de|ru|ko|ar|id|nl|pl|tr|th|uk|cs|sv|da|fi|no|he|hi|bn'

lowercase() {
	printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

# Localized documentation names look like <stem>.<locale>.md or
# <stem>_<locale>.md, in any letter case.
is_localized_doc() {
	local base

	base="$(lowercase "$(basename "$1")")"

	[[ "$base" =~ \.(${locale_codes})\.md$ ]] && return 0
	[[ "$base" =~ _(${locale_codes})\.md$ ]] && return 0

	return 1
}

# The English counterpart of a localized document: same directory, locale
# marker stripped. Matching is case-insensitive; the suggestion keeps the
# original filename casing.
english_sibling() {
	local path="$1"
	local dir base lower stem marker

	dir="$(dirname "$path")"
	base="$(basename "$path")"
	lower="$(lowercase "$base")"
	stem="${base%.md}"

	if [[ "$lower" =~ ^(.+)\.(${locale_codes})\.md$ ]]; then
		marker="${BASH_REMATCH[2]}"
		stem="${stem:0:$((${#stem} - ${#marker} - 1))}"
	elif [[ "$lower" =~ ^(.+)_(${locale_codes})\.md$ ]]; then
		marker="${BASH_REMATCH[2]}"
		stem="${stem:0:$((${#stem} - ${#marker} - 1))}"
	fi

	printf '%s/%s.md' "$dir" "$stem"
}

# Language buckets such as docs/zh/, docs/guides/zh/, or
# docs/channels/telegram/zh/ are not allowed anywhere under docs/.
is_locale_directory() {
	local path="$1"
	local segment
	local -a segments

	[[ "$path" == docs/* ]] || return 1

	IFS='/' read -r -a segments <<<"$(lowercase "${path#docs/}")"
	for segment in "${segments[@]}"; do
		[[ "|${locale_codes}|" == *"|${segment}|"* ]] && return 0
	done

	return 1
}

while IFS= read -r path; do
	[[ -f "$path" ]] || continue

	# Localized documentation is rejected anywhere in the repository, not only
	# under docs/: one canonical English file per document.
	if is_localized_doc "$path"; then
		english="$(english_sibling "$path")"

		error \
			"$path" \
			"documentation is English-only; localized variants are not kept" \
			"delete this file and keep only '${english}'"
	fi

	if is_locale_directory "$path"; then
		error \
			"$path" \
			"documentation is English-only; language bucket directories are not allowed" \
			"move the file to its typed English directory, for example docs/guides/"
	fi

	# docs/AGENTS.md is a required DOX file, so exempt the DOX filename by
	# basename instead of moving it into a typed subdirectory.
	if [[ "$path" =~ ^docs/[^/]+\.md$ && "$path" != "docs/README.md" && "$(basename "$path")" != "AGENTS.md" ]]; then
		error \
			"$path" \
			"top-level docs Markdown files must move into a typed docs/ subdirectory" \
			"move it into one of docs/guides/, docs/reference/, docs/operations/, docs/security/, docs/architecture/, docs/channels/, docs/design/, or docs/migration/"
	fi
done < <(git ls-files --cached --others --exclude-standard -- '*.md')

if [[ "$failures" -ne 0 ]]; then
	echo "docs lint: failed" >&2
	exit 1
fi

echo "docs lint: OK"
