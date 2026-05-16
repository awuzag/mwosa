#!/bin/sh
set -eu

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

go_cmd="${GO:-go}"

run_step() {
	name="$1"
	shift
	printf "\n==> %s\n" "$name" >&2
	"$@"
}

check_go_format() {
	files="$(git ls-files '*.go')"
	if [ -z "$files" ]; then
		return 0
	fi
	unformatted="$(gofmt -l $files)"
	if [ -n "$unformatted" ]; then
		printf "%s\n" "gofmt required:" >&2
		printf "%s\n" "$unformatted" >&2
		return 1
	fi
}

test_client_modules() {
	for module in \
		clients/datago-corpfin \
		clients/datago-etp \
		clients/datago-krxlisted \
		clients/datago-stock-price \
		clients/kis \
		clients/krx
	do
		printf "\n==> %s\n" "$module" >&2
		(cd "$module" && "$go_cmd" test ./... && "$go_cmd" mod verify)
	done
}

run_step "gofmt check" check_go_format
run_step "root go test" "$go_cmd" test ./...
run_step "provider client module tests" test_client_modules
run_step "diff whitespace check" git diff --check

printf "\npre-commit checks passed\n" >&2
