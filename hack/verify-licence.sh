#!/bin/bash
#
# Description: is used to check all the files have the same headers

#
BOILERPLATE=${BOILERPLATE:-"hack/boilerplate.go.txt"}
BOILERPLATE_LENGTH=$(cat ${BOILERPLATE}| wc -l | xargs)
EXCLUDE_FILES=(
)

if [[ -z "${BOILERPLATE_LENGTH}" ]]; then
  echo "Failed to retrieve length of header in ${BOIERPLATE}"
  exit 1
fi

failed=0
while read name; do
  # ignore excluded files
  [[ " ${EXCLUDE_FILES[*]} " == *" ${name} "* ]] && continue
  # ignore auto generated ones
  [[ "${name}" =~ ^.*zz_generated.*$ ]] && continue
  # ignore test suite files
  [[ "${name}" =~ ^.*_suite_test.go$ ]] && continue
  # ignore generated files
  if head -n 1 "${name}" | grep -qE "^// Code generated"; then
    continue
  fi
  # skip build tags when checking
  skipLines=0
  if head -n 1 "${name}" | grep -qE "^// \+build"; then
    skipLines=3
  fi

  if ! tail -n "+$skipLines" ${name} | head -n ${BOILERPLATE_LENGTH} | diff - ${BOILERPLATE} >/dev/null; then
    echo "Missing licence header: ${name}"
    failed=1
  fi
done < <(find . -type f -name "*.go" | grep -v vendor)

if [ "$failed" = 1 ]; then
  echo
  echo "Make sure all listed files have a licence header. The licence can be found in ${BOILERPLATE}".
  echo
  echo "# Copy to clipboard:"
  echo "cat ${BOILERPLATE} | pbcopy"
  echo
  exit 1
fi
