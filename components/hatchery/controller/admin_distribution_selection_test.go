package controller

import (
	"slices"
	"testing"
)

func TestDistributionSelectionValidate(t *testing.T) {
	tests := []struct {
		name      string
		selection distributionSelection
		wantErr   bool
	}{
		{name: "explicit IDs", selection: distributionSelection{InstanceIDs: []uint{1}}},
		{name: "select all", selection: distributionSelection{SelectAll: true}},
		{name: "both modes", selection: distributionSelection{InstanceIDs: []uint{1}, SelectAll: true}, wantErr: true},
		{name: "neither mode", selection: distributionSelection{}, wantErr: true},
		{name: "statuses without select all", selection: distributionSelection{InstanceIDs: []uint{1}, Statuses: []string{"failed"}}, wantErr: true},
		{name: "groups without select all", selection: distributionSelection{InstanceIDs: []uint{1}, GroupIDs: []uint{1}}, wantErr: true},
		{name: "search without select all", selection: distributionSelection{InstanceIDs: []uint{1}, Search: "agent"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.selection.validate()
			if gotErr := err != nil; gotErr != tt.wantErr {
				t.Errorf("distributionSelection.validate(%+v) error = %v, want error presence = %t", tt.selection, err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeDistributionStatuses(t *testing.T) {
	allowed := []string{"uninstalled", "installed", "failed"}
	transitional := []string{"installing"}

	got, err := normalizeDistributionStatuses(nil, allowed, transitional)
	if err != nil {
		t.Fatalf("normalizeDistributionStatuses(nil) error = %v", err)
	}
	if !slices.Equal(got, allowed) {
		t.Fatalf("normalizeDistributionStatuses(nil) = %v, want %v", got, allowed)
	}
	got[0] = "changed"
	if allowed[0] != "uninstalled" {
		t.Fatal("normalizeDistributionStatuses(nil) must return a copy of allowed statuses")
	}

	got, err = normalizeDistributionStatuses([]string{"failed", "uninstalled", "failed"}, allowed, transitional)
	if err != nil {
		t.Fatalf("normalizeDistributionStatuses(deduplicate) error = %v", err)
	}
	if want := []string{"failed", "uninstalled"}; !slices.Equal(got, want) {
		t.Fatalf("normalizeDistributionStatuses(deduplicate) = %v, want %v", got, want)
	}

	if _, err := normalizeDistributionStatuses([]string{"installing"}, allowed, transitional); err == nil {
		t.Fatal("normalizeDistributionStatuses(transitional) error = nil, want error")
	}
	if _, err := normalizeDistributionStatuses([]string{"typo"}, allowed, transitional); err == nil {
		t.Fatal("normalizeDistributionStatuses(invalid) error = nil, want error")
	}
}
