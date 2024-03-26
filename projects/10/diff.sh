#!/bin/bash -x

file="$1"
[ -z "${file}" ] && exit 1

dst="$(dirname "${file}")/$(basename "${file}" ".xml").xmlf"

diff -urN --strip-trailing-cr "${file}" "${dst}"
