package common

import "testing"

func TestIsUniverseMode_WhenFixedSnapshotNil(t *testing.T) {
	old := FixedSnapshot
	defer func() { FixedSnapshot = old }()

	FixedSnapshot = nil
	if !IsUniverseMode() {
		t.Fatal("expected true when FixedSnapshot is nil")
	}
}

func TestIsUniverseMode_WhenFixedSnapshotSet(t *testing.T) {
	old := FixedSnapshot
	defer func() { FixedSnapshot = old }()

	FixedSnapshot = &TenantSnapshot{Identifier: "test"}
	if IsUniverseMode() {
		t.Fatal("expected false when FixedSnapshot is not nil")
	}
}

func TestIsUniverseMode_EmptyIdentifier(t *testing.T) {
	old := FixedSnapshot
	defer func() { FixedSnapshot = old }()

	// SQLite 本地模式：FixedSnapshot 存在但 Identifier 为空
	FixedSnapshot = &TenantSnapshot{Identifier: ""}
	if IsUniverseMode() {
		t.Fatal("expected false when FixedSnapshot is set (even with empty identifier)")
	}
}
