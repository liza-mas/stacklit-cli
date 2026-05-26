#!/bin/bash
set -euo pipefail

if command -v scip-go >/dev/null ; then
  echo "SCIP Indexing..."
  scip-go index --module-root "$(pwd)" --output go.scip
  echo "Wrote go.scip"
  echo
fi

if command -v stacklit >/dev/null ; then
  code=0
  if [ -f stacklit.json ] ; then
    stacklit diff >/dev/null || code=$?
    case "$code" in
      0) [ "${1:-}" = "ai" ] || exit 0 ;;
      1) ;;
      *) echo "stacklit diff failed"; exit "$code" ;;
    esac
  fi

  echo "Stacklit Indexing..."
  stacklit generate-json
  stacklit init-insights
  if [ "${1:-}" = "ai" ]
  then
    echo "Adding AI summary..."
    stacklit ai-summary
  fi
  stacklit generate-json
  echo "Wrote stacklit.json"
fi
