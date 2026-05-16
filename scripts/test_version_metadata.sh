#!/bin/sh
set -eu

expected_version="$(git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//')"
if [ -z "$expected_version" ]; then
	expected_version="dev"
fi

expected_commit="$(git rev-parse --short HEAD 2>/dev/null || true)"
if [ -z "$expected_commit" ]; then
	expected_commit="none"
fi

make_database="$(
	unset VERSION COMMIT
	make -pn build
)"

ldflags="$(printf '%s\n' "$make_database" | awk -F' := ' '$1 == "LDFLAGS" { print $2; exit }')"
actual_version="$(printf '%s\n' "$ldflags" | sed -n 's/.*cmd\.version=\([^ ]*\).*/\1/p')"
actual_commit="$(printf '%s\n' "$ldflags" | sed -n 's/.*cmd\.commit=\([^ ]*\).*/\1/p')"

if [ "$actual_version" != "$expected_version" ]; then
	printf 'VERSION=%s, want %s\n' "$actual_version" "$expected_version" >&2
	exit 1
fi

if [ "$actual_commit" != "$expected_commit" ]; then
	printf 'COMMIT=%s, want %s\n' "$actual_commit" "$expected_commit" >&2
	exit 1
fi
