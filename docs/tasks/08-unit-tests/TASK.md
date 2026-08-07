# Task 8: Add unit tests for core pure functions

**Status:** done (folded into other tasks)

- [x] videoid + format tests done as part of task 2
- [x] `parseVtt` + transcript formatting/error tests done as part of task 4
- [x] `qualityFormat` + download message templates done as part of task 5
- [x] Fuzz tests for the highest-value untrusted-input parsers done as a
      separate out-of-band item — see `docs/tasks/fuzz-tests/TASK.md`
- [ ] Remaining: cobra/mcpserver-level tests if any pure logic emerges there
      (most of tasks 6–7 is I/O wiring, likely covered by the task 9 smoke
      test instead)

This task never needed its own dedicated work session — every pure function
introduced by tasks 2–5 got its test as part of that task under the
project's TDD process (see `CLAUDE.md`), so there was nothing left over to
do here specifically.
