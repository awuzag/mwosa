#!/bin/sh

set -eu

root=$(git rev-parse --show-toplevel)
branch=$(git branch --show-current)
if [ -z "$branch" ]; then
	branch="detached-$(git rev-parse --short HEAD)"
fi

sanitize() {
	printf '%s' "$1" |
		tr '[:upper:]' '[:lower:]' |
		sed 's/[^a-z0-9_]/-/g; s/--*/-/g; s/^[-_]*//; s/[-_]*$//'
}

hash_text() {
	if command -v shasum >/dev/null 2>&1; then
		shasum -a 256 | awk '{print substr($1, 1, 10)}'
		return
	fi
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum | awk '{print substr($1, 1, 10)}'
		return
	fi
	echo "shasum or sha256sum is required" >&2
	exit 1
}

validate_database() {
	value=$1
	if [ -z "$value" ]; then
		echo "MongoDB database name is required" >&2
		exit 1
	fi
	case "$value" in
	*[!A-Za-z0-9_-]*)
		echo "MongoDB database name must contain only letters, numbers, hyphens, or underscores: $value" >&2
		exit 1
		;;
	esac
	if [ "${#value}" -gt 63 ]; then
		echo "MongoDB database name must be at most 63 characters: $value" >&2
		exit 1
	fi
}

database_from_url() {
	value=$1
	case "$value" in
	mongodb://* | mongodb+srv://*) ;;
	*)
		echo "MWOSA_DATABASE_URL must be a MongoDB URL" >&2
		exit 1
		;;
	esac
	without_query=${value%%\?*}
	authority_and_path=${without_query#*://}
	case "$authority_and_path" in
	*/*)
		value_database=${authority_and_path#*/}
		;;
	*)
		echo "MWOSA_DATABASE_URL must include a database name" >&2
		exit 1
		;;
	esac
	case "$value_database" in
	'' | */*)
		echo "MWOSA_DATABASE_URL must contain one database path segment" >&2
		exit 1
		;;
	esac
	printf '%s\n' "$value_database"
}

compose_url() {
	server=$1
	value_database=$2
	case "$server" in
	mongodb://* | mongodb+srv://*) ;;
	*)
		echo "MWOSA_MONGODB_URI must be a MongoDB server URL" >&2
		exit 1
		;;
	esac
	query=""
	case "$server" in
	*\?*) query="?${server#*\?}" ;;
	esac
	base=${server%%\?*}
	base=${base%/}
	authority_and_path=${base#*://}
	case "$authority_and_path" in
	*/*)
		echo "MWOSA_MONGODB_URI must not include a database path; use MWOSA_DATABASE_URL for a full URL" >&2
		exit 1
		;;
	esac
	printf '%s/%s%s\n' "$base" "$value_database" "$query"
}

redact_url() {
	printf '%s' "$1" | sed 's#://[^/@]*@#://<redacted>@#'
}

worktree_id=$(printf '%s' "$root" | hash_text)
repo_name=$(sanitize "$(basename "$root")")
if [ -f "$root/.git" ]; then
	worktree_name=$(sanitize "$(basename "$(dirname "$root")")")
	default_database="$worktree_name-$repo_name"
else
	default_database=$repo_name
fi
if [ -z "$default_database" ]; then
	default_database="mwosa-$worktree_id"
fi
if [ "${#default_database}" -gt 63 ]; then
	database_prefix=$(printf '%s' "$default_database" | cut -c '1-52' | sed 's/[-_]*$//')
	default_database="$database_prefix-$worktree_id"
fi

full_url=${MWOSA_DATABASE_URL:-}
server_uri=${MWOSA_MONGODB_URI:-}
database_override=${MWOSA_MONGODB_DATABASE:-}
if [ -n "$full_url" ] && { [ -n "$server_uri" ] || [ -n "$database_override" ]; }; then
	echo "set either MWOSA_DATABASE_URL or MWOSA_MONGODB_URI/MWOSA_MONGODB_DATABASE, not both" >&2
	exit 1
fi

managed=true
mode="managed-container"
database=${database_override:-$default_database}
configured_url=""
if [ -n "$full_url" ]; then
	managed=false
	mode="full-url"
	database=$(database_from_url "$full_url")
	configured_url=$full_url
elif [ -n "$server_uri" ]; then
	managed=false
	mode="shared-server"
	validate_database "$database"
	configured_url=$(compose_url "$server_uri" "$database")
else
	validate_database "$database"
fi

container="mwosa-mongodb-$worktree_id"
volume="mwosa-mongodb-$worktree_id-data"
image=${MONGODB_IMAGE:-mongo:7}
worktree_number=$(printf '%d' "0x$(printf '%s' "$worktree_id" | cut -c '1-6')")
host_port=${MONGODB_PORT:-$((20000 + worktree_number % 20000))}
case "$host_port" in
'' | *[!0-9]*)
	echo "MONGODB_PORT must be a number" >&2
	exit 1
	;;
esac
if [ "$host_port" -lt 1 ] || [ "$host_port" -gt 65535 ]; then
	echo "MONGODB_PORT must be between 1 and 65535" >&2
	exit 1
fi

require_docker() {
	if ! command -v docker >/dev/null 2>&1; then
		echo "docker is required" >&2
		exit 1
	fi
	if ! docker info >/dev/null 2>&1; then
		echo "Docker daemon is not running" >&2
		exit 1
	fi
}

is_running() {
	[ "$(docker inspect -f '{{.State.Running}}' "$container" 2>/dev/null || true)" = "true" ]
}

port() {
	if ! is_running; then
		echo "MongoDB container is not running: $container" >&2
		exit 1
	fi
	docker port "$container" 27017/tcp | awk -F: 'END {print $NF}'
}

uri() {
	if [ "$managed" = true ]; then
		printf 'mongodb://127.0.0.1:%s/%s\n' "$(port)" "$database"
		return
	fi
	printf '%s\n' "$configured_url"
}

display_uri() {
	redact_url "$(uri)"
	printf '\n'
}

wait_until_ready() {
	attempt=0
	while [ "$attempt" -lt 30 ]; do
		if docker exec "$container" mongosh --quiet --eval 'quit(db.runCommand({ping: 1}).ok ? 0 : 1)' >/dev/null 2>&1; then
			return
		fi
		attempt=$((attempt + 1))
		sleep 1
	done
	echo "MongoDB did not become ready within 30 seconds: $container" >&2
	exit 1
}

up() {
	if [ "$managed" != true ]; then
		printf 'Using externally managed MongoDB\nmode: %s\ndatabase: %s\nurl: %s\n' \
			"$mode" "$database" "$(redact_url "$configured_url")"
		return
	fi
	require_docker
	if docker inspect "$container" >/dev/null 2>&1; then
		current_port=$(docker port "$container" 27017/tcp 2>/dev/null | awk -F: 'END {print $NF}')
		if [ "$current_port" != "$host_port" ]; then
			docker rm -f "$container" >/dev/null
		elif ! is_running; then
			docker start "$container" >/dev/null
		fi
	fi
	if ! docker inspect "$container" >/dev/null 2>&1; then
		docker run -d \
			--name "$container" \
			--label "com.awuzag.mwosa.worktree=$root" \
			--mount "type=volume,source=$volume,target=/data/db" \
			--publish "127.0.0.1:$host_port:27017" \
			"$image" >/dev/null
	fi
	wait_until_ready
	printf 'MongoDB ready\nmode: %s\ncontainer: %s\ndatabase: %s\nurl: %s\n' \
		"$mode" "$container" "$database" "$(display_uri)"
}

down() {
	if [ "$managed" != true ]; then
		printf 'MongoDB lifecycle is externally managed; nothing was stopped.\nmode: %s\nurl: %s\n' \
			"$mode" "$(redact_url "$configured_url")"
		return
	fi
	require_docker
	if ! docker inspect "$container" >/dev/null 2>&1; then
		printf 'MongoDB container does not exist: %s\n' "$container"
		return
	fi
	if is_running; then
		docker stop "$container" >/dev/null
	fi
	printf 'MongoDB stopped; data volume preserved: %s\n' "$volume"
}

reset() {
	if [ "$managed" != true ]; then
		echo "refusing to reset externally managed MongoDB" >&2
		exit 1
	fi
	require_docker
	if docker inspect "$container" >/dev/null 2>&1; then
		docker rm -f "$container" >/dev/null
	fi
	if docker volume inspect "$volume" >/dev/null 2>&1; then
		docker volume rm "$volume" >/dev/null
	fi
	printf 'MongoDB container and data removed for worktree: %s\n' "$root"
}

status() {
	printf 'worktree: %s\nbranch: %s\nworktree_id: %s\nmode: %s\ndatabase: %s\n' \
		"$root" "$branch" "$worktree_id" "$mode" "$database"
	if [ "$managed" != true ]; then
		printf 'url: %s\n' "$(redact_url "$configured_url")"
		return
	fi
	state="missing"
	url=""
	if command -v docker >/dev/null 2>&1 && docker inspect "$container" >/dev/null 2>&1; then
		state="stopped"
		if is_running; then
			state="running"
			url=$(display_uri)
		fi
	fi
	printf 'container: %s\ncontainer_state: %s\n' "$container" "$state"
	if [ -n "$url" ]; then
		printf 'url: %s\n' "$url"
	fi
}

case "${1:-status}" in
up)
	up
	;;
down)
	down
	;;
reset)
	reset
	;;
status)
	status
	;;
uri)
	if [ "$managed" = true ]; then
		require_docker
	fi
	uri
	;;
display-uri)
	if [ "$managed" = true ]; then
		require_docker
	fi
	display_uri
	;;
*)
	echo "usage: $0 {up|down|reset|status|uri|display-uri}" >&2
	exit 2
	;;
esac
