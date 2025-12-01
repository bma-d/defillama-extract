# Engineering Backlog

This backlog collects cross-cutting or future action items that emerge from reviews and planning.

Routing guidance:

- Use this file for non-urgent optimizations, refactors, or follow-ups that span multiple stories/epics.
- Must-fix items to ship a story belong in that story’s `Tasks / Subtasks`.
- Same-epic improvements may also be captured under the epic Tech Spec `Post-Review Follow-ups` section.

| Date | Story | Epic | Type | Severity | Owner | Status | Notes |
| ---- | ----- | ---- | ---- | -------- | ----- | ------ | ----- |
| 2025-12-01 | 5-1-implement-output-file-generation | 5 | Bug | High | TBD | Closed | Writer now honors cfg.Output.{FullFile,MinFile,SummaryFile}; tests assert custom filenames (internal/storage/writer.go, internal/storage/writer_test.go). |
| 2025-12-01 | 5-1-implement-output-file-generation | 5 | Bug | Medium | TBD | Closed | metadata.update_frequency now mirrors cfg.Scheduler.Interval with defaults; tests cover scheduler interval string (internal/storage/writer.go, internal/storage/writer_test.go). |
