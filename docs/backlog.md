# Engineering Backlog

This backlog collects cross-cutting or future action items that emerge from reviews and planning.

Routing guidance:

- Use this file for non-urgent optimizations, refactors, or follow-ups that span multiple stories/epics.
- Must-fix items to ship a story belong in that story’s `Tasks / Subtasks`.
- Same-epic improvements may also be captured under the epic Tech Spec `Post-Review Follow-ups` section.

| Date | Story | Epic | Type | Severity | Owner | Status | Notes |
| ---- | ----- | ---- | ---- | -------- | ----- | ------ | ----- |
| 2025-12-03 | 5-4-extract-historical-chart-data | 5 | Advisory | Low | TBD | Open | Monitor `chart_history` size and write time as history grows (current: 1,466 points, 2021-11-29..2025-12-03 in `data/switchboard-oracle-data.json`); consider compression or streaming write if latency degrades. |
| 2025-12-02 | 5-3-implement-daemon-mode | 5 | Bug | High | Amelia | Closed | SIGINT in `--once` can still persist outputs because `WriteAllOutputs` is not context-aware; add ctx-gated writes to satisfy AC8 (`cmd/extractor/main.go:170-190`, `internal/storage/writer.go:158-190`). |
| 2025-12-02 | 5-3-implement-daemon-mode | 5 | Test Gap | Med | Amelia | Closed | No test covers SIGINT during `WriteAllOutputs` to assert no files are created and exit code is 1; add once-mode cancellation test in `cmd/extractor/main_test.go`. |
| 2025-12-02 | 5-3-implement-daemon-mode | 5 | Bug | High | Amelia | Closed | `RunOnce` allows writes/state after SIGINT in `--once`; add cancellation guard between generation and writes and exit 1 on cancellation (cmd/extractor/main.go:167-195). |
| 2025-12-02 | 5-3-implement-daemon-mode | 5 | Test Gap | Med | TBD | Open | Missing integration/smoke tests for daemon scheduling and signal handling (start_immediately true/false, shutdown during wait/run, error recovery); add short-interval coverage. |
| 2025-12-02 | 5-2-implement-cli-and-single-run-mode | 5 | Bug | Med | TBD | Open | Daemon start_immediately failures still exit instead of logging and waiting for the next interval; update `runDaemonWithDeps` to continue scheduling (cmd/extractor/main.go:222-225). |
| 2025-12-01 | 5-1-implement-output-file-generation | 5 | Bug | High | TBD | Closed | Writer now honors cfg.Output.{FullFile,MinFile,SummaryFile}; tests assert custom filenames (internal/storage/writer.go, internal/storage/writer_test.go). |
| 2025-12-01 | 5-1-implement-output-file-generation | 5 | Bug | Medium | TBD | Closed | metadata.update_frequency now mirrors cfg.Scheduler.Interval with defaults; tests cover scheduler interval string (internal/storage/writer.go, internal/storage/writer_test.go). |
| 2025-12-01 | 5-2-implement-cli-and-single-run-mode | 5 | Bug | High | TBD | Closed | `--dry-run` no longer persists state/history; guarded in `cmd/extractor/main.go:136-149` and covered by `cmd/extractor/main_test.go:229-266`. |
| 2025-12-01 | 5-2-implement-cli-and-single-run-mode | 5 | Bug | Medium | TBD | Closed | CLI parse errors now bubble up with exit code 2 and tests in `cmd/extractor/main_test.go:119-133,269-283`. |
| 2025-12-01 | 5-2-implement-cli-and-single-run-mode | 5 | Test Gap | Medium | TBD | Closed | `internal/api/responses_test.go:104-122` covers the `protocols` envelope regression handled by `internal/api/responses.go:30-58`. |
| 2025-12-01 | 5-2-implement-cli-and-single-run-mode | 5 | Bug | High | TBD | Closed | Default execution without flags now starts daemon ticker with signal-aware context; covered by `runDaemonWithDeps` and tests (`cmd/extractor/main.go`, `cmd/extractor/main_test.go:250-310`). |
