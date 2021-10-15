#!/usr/bin/env bash
# write-dockertagfile: write-ifchanged, but keeps Docker tags in-sync
#
# Copyright (C) 2015  Luke Shumaker
# Copyright 2019 Datawire
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <http://www.gnu.org/licenses/>.

outfile=$1
tmpfile="$(dirname "$outfile")/.tmp.${outfile##*/}.tmp"

cat > "$tmpfile" || exit $?
if cmp -s "$tmpfile" "$outfile"; then
	rm -f "$tmpfile" || :
else
	# It's a little bit tempting to try to merge this "-e
	# $outfile" check with the one for "docker image rm".  Don't.
	# We do the CI check early, before any destructive changes
	# have been made.  We do the rm check late, so that that any
	# image IDs pointed to by tag names that haven't changed are
	# un-pinned by then, and we don't leak orphaned images.
	set -e
	IFS=''
	{
		read -r iid
		while read -r tag; do
			docker image tag -- "$iid" "$tag"
		done
	} < "$tmpfile"
	if [[ -e "$outfile" ]]; then
		args="$(grep -vFx -f "$tmpfile" -- "$outfile")"
		if [[ -n "$args" ]]; then
			docker image rm -- $args || :
		fi
	fi
	mv -f "$tmpfile" "$outfile"
fi
