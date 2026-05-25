#!/bin/sh
set -eu

REPO_OWNER=${STACKLIT_REPO_OWNER:-liza-mas}
REPO_NAME=${STACKLIT_REPO_NAME:-stacklit-cli}
BRANCH=${BRANCH:-main}
INSTALL_DIR=${INSTALL_DIR:-"$HOME/.local/bin"}
SOURCE_REPO=${STACKLIT_SOURCE_REPO:-"https://github.com/${REPO_OWNER}/${REPO_NAME}.git"}
SOURCE_TMPDIR=${STACKLIT_SOURCE_TMPDIR:-"${TMPDIR:-/tmp}"}

fail() {
	printf 'stacklit install: %s\n' "$*" >&2
	exit 1
}

install_branch() {
	source_branch=$1

	command -v go >/dev/null 2>&1 || fail "Go is required for branch builds"
	command -v make >/dev/null 2>&1 || fail "make is required for branch builds"
	command -v git >/dev/null 2>&1 || fail "git is required to fetch BRANCH=$source_branch"

	source_root=$(mktemp -d "${SOURCE_TMPDIR%/}/stacklit-source.XXXXXX") || fail "could not create temporary source checkout"
	cleanup_source_root() {
		rm -rf "$source_root"
	}
	trap cleanup_source_root EXIT HUP INT TERM

	source_dir=$source_root/src
	if ! git clone --quiet --depth 1 --branch "$source_branch" "$SOURCE_REPO" "$source_dir"; then
		fail "BRANCH=$source_branch is unavailable from $SOURCE_REPO"
	fi

	if ! source_revision=$(git -C "$source_dir" rev-parse --short HEAD 2>/dev/null); then
		fail "could not identify source revision for BRANCH=$source_branch"
	fi

	if ! make -C "$source_dir" install INSTALL_DIR="$INSTALL_DIR" SOURCE_REF="branch:$source_branch" SOURCE_REVISION="$source_revision"; then
		fail "source install failed for BRANCH=$source_branch"
	fi

	installed_path=$INSTALL_DIR/stacklit
	if [ ! -x "$installed_path" ]; then
		fail "source install did not create executable stacklit at $installed_path"
	fi

	if ! version_output=$("$installed_path" --version 2>&1); then
		fail "installed stacklit at $installed_path failed --version for BRANCH=$source_branch"
	fi
	case "$version_output" in
	*"branch:$source_branch"*"$source_revision"*)
		;;
	*)
		fail "installed stacklit at $installed_path did not report source provenance for BRANCH=$source_branch"
		;;
	esac

	printf 'Installed stacklit source branch=%s revision=%s to %s\n' "$source_branch" "$source_revision" "$installed_path"
}

install_branch "$BRANCH"
