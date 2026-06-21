#!/usr/bin/env bash
# Gate stub for the canonical review.af flow.
#
# review.af's `quality` gate runs `scripts/test.sh` relative to the project
# root where the emitted host config lives (the directory that contains
# .cursor/ or .claude/). The happy path simply succeeds; a real project would
# run its test suite here and exit non-zero to trip the gate's retry policy.
set -euo pipefail
exit 0
