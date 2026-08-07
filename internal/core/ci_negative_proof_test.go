package core

import "testing"

// This test exists only to prove CI actually goes red on a real failure —
// see docs/tasks/ci-cd/TASK.md. It is deliberately broken and this whole
// file is deleted before merging.
func TestCINegativeProofDeliberatelyFails(t *testing.T) {
	t.Fatal("intentional failure to prove CI reports red — see docs/tasks/ci-cd/TASK.md")
}
