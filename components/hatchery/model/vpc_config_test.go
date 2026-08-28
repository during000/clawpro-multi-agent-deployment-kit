package model

import (
	"testing"
)

func TestGetSubnetMap_Empty(t *testing.T) {
	v := VpcConfig{SubnetIds: ""}
	m, err := v.GetSubnetMap()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestGetSubnetMap_NullJSON(t *testing.T) {
	v := VpcConfig{SubnetIds: "null"}
	m, err := v.GetSubnetMap()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil map")
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestGetSubnetMap_Valid(t *testing.T) {
	v := VpcConfig{SubnetIds: `{"ap-guangzhou-3":["subnet-abc","subnet-def"],"ap-guangzhou-4":["subnet-xyz"]}`}
	m, err := v.GetSubnetMap()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 2 {
		t.Errorf("expected 2 zones, got %d", len(m))
	}
	if len(m["ap-guangzhou-3"]) != 2 {
		t.Errorf("expected 2 subnets in zone 3, got %d", len(m["ap-guangzhou-3"]))
	}
}

func TestGetSubnetMap_InvalidJSON(t *testing.T) {
	v := VpcConfig{SubnetIds: "not-json"}
	_, err := v.GetSubnetMap()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
