#!/bin/sh

set -eu

script="scripts/dev/worktree-mongodb.sh"

managed=$(MWOSA_DATABASE_URL= MWOSA_MONGODB_URI= MWOSA_MONGODB_DATABASE= sh "$script" status)
case "$managed" in
*"mode: managed-container"*) ;;
*)
	echo "managed container mode was not selected" >&2
	exit 1
	;;
esac

full_url="mongodb://user:secret@db.example:27018/custom-db?authSource=admin"
full_status=$(MWOSA_DATABASE_URL="$full_url" MWOSA_MONGODB_URI= MWOSA_MONGODB_DATABASE= sh "$script" status)
case "$full_status" in
*"mode: full-url"*"database: custom-db"*"mongodb://<redacted>@db.example:27018/custom-db?authSource=admin"*) ;;
*)
	echo "full URL mode did not preserve the database or redact credentials" >&2
	exit 1
	;;
esac
case "$full_status" in
*secret*)
	echo "full URL status exposed credentials" >&2
	exit 1
	;;
esac
full_raw=$(MWOSA_DATABASE_URL="$full_url" MWOSA_MONGODB_URI= MWOSA_MONGODB_DATABASE= sh "$script" uri)
if [ "$full_raw" != "$full_url" ]; then
	echo "full URL mode changed the internal connection URL" >&2
	exit 1
fi

split_url=$(MWOSA_DATABASE_URL= MWOSA_MONGODB_URI="mongodb://127.0.0.1:27017" MWOSA_MONGODB_DATABASE="aggregate-mwosa" sh "$script" uri)
if [ "$split_url" != "mongodb://127.0.0.1:27017/aggregate-mwosa" ]; then
	echo "server and database mode produced an unexpected URL: $split_url" >&2
	exit 1
fi

if MWOSA_DATABASE_URL="$full_url" MWOSA_MONGODB_URI="mongodb://127.0.0.1:27017" MWOSA_MONGODB_DATABASE= sh "$script" status >/dev/null 2>&1; then
	echo "ambiguous MongoDB configuration was accepted" >&2
	exit 1
fi

if MWOSA_DATABASE_URL= MWOSA_MONGODB_URI="mongodb://127.0.0.1:27017/existing" MWOSA_MONGODB_DATABASE="aggregate-mwosa" sh "$script" uri >/dev/null 2>&1; then
	echo "server URL with a database path was accepted" >&2
	exit 1
fi

printf 'worktree MongoDB configuration tests passed\n'
