package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunScriptsRunsExclusivePrefixScriptsSeparately(t *testing.T) {
	tempDir := t.TempDir()
	activeFile := filepath.Join(tempDir, "regular-active")
	completedFile := filepath.Join(tempDir, "exclusive-completed")
	regularScript := filepath.Join(tempDir, "test_regular.py")
	exclusiveScript := filepath.Join(tempDir, "test_exclusive_shared_fixture.py")

	regularSource := `import os, pathlib, time
active = pathlib.Path(os.environ["ACTIVE_FILE"])
active.write_text("active")
time.sleep(0.2)
active.unlink()
`
	exclusiveSource := `import os, pathlib, sys
active = pathlib.Path(os.environ["ACTIVE_FILE"])
if active.exists():
    sys.exit(7)
pathlib.Path(os.environ["COMPLETED_FILE"]).write_text("done")
`
	if err := os.WriteFile(regularScript, []byte(regularSource), 0o600); err != nil {
		t.Fatalf("write regular script: %v", err)
	}
	if err := os.WriteFile(exclusiveScript, []byte(exclusiveSource), 0o600); err != nil {
		t.Fatalf("write exclusive script: %v", err)
	}

	results := runScripts(
		context.Background(),
		[]string{exclusiveScript, regularScript},
		[]string{
			fmt.Sprintf("ACTIVE_FILE=%s", activeFile),
			fmt.Sprintf("COMPLETED_FILE=%s", completedFile),
		},
		2,
		5*time.Second,
	)
	if len(results) != 2 {
		t.Fatalf("result count=%d, want 2", len(results))
	}
	for _, result := range results {
		if result.err != nil {
			t.Fatalf("script %s failed: %v\n%s", filepath.Base(result.script), result.err, result.output)
		}
	}
	if _, err := os.Stat(completedFile); err != nil {
		t.Fatalf("exclusive script did not complete: %v", err)
	}
}
