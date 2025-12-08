package tvl

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTVLStateManagerLoadMissing(t *testing.T) {
	dir := t.TempDir()
	mgr := NewTVLStateManager(dir, nil)

	state, err := mgr.LoadState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.LastUpdated != 0 {
		t.Fatalf("expected zero state, got %+v", state)
	}
}

func TestTVLStateManagerSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	mgr := NewTVLStateManager(dir, nil)

	in := &TVLState{LastUpdated: 100, ProtocolCount: 2, CustomProtocolCnt: 1}
	if err := mgr.SaveState(in); err != nil {
		t.Fatalf("save state failed: %v", err)
	}

	path := filepath.Join(dir, "tvl-state.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected state file, got error: %v", err)
	}

	out, err := mgr.LoadState()
	if err != nil {
		t.Fatalf("load state failed: %v", err)
	}
	if out.LastUpdated != 100 || out.ProtocolCount != 2 || out.CustomProtocolCnt != 1 {
		t.Fatalf("unexpected loaded state: %+v", out)
	}
}

func TestTVLShouldProcess(t *testing.T) {
	mgr := NewTVLStateManager(t.TempDir(), nil)
	state := &TVLState{LastUpdated: 10}

	if !mgr.ShouldProcess(20, state) {
		t.Fatalf("expected process when newer timestamp")
	}
	if mgr.ShouldProcess(10, state) {
		t.Fatalf("expected skip when timestamp equal")
	}
}
