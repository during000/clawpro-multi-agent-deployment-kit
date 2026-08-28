package model

import (
	"context"
	"testing"
)

func TestFilterInstancesByUserGroups_UnionAndUngrouped(t *testing.T) {
	db := setupSeedTestDB(t)
	ctx := context.Background()
	groupA := UserGroup{Name: "select-all-group-a"}
	groupB := UserGroup{Name: "select-all-group-b"}
	if err := db.Create(&groupA).Error; err != nil {
		t.Fatalf("create group A: %v", err)
	}
	if err := db.Create(&groupB).Error; err != nil {
		t.Fatalf("create group B: %v", err)
	}
	users := []User{{Username: "select-all-a"}, {Username: "select-all-b"}, {Username: "select-all-none"}}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	if err := db.Create(&[]UserGroupMember{
		{UserID: users[0].ID, UserGroupID: groupA.ID},
		{UserID: users[0].ID, UserGroupID: groupB.ID},
		{UserID: users[1].ID, UserGroupID: groupB.ID},
	}).Error; err != nil {
		t.Fatalf("create memberships: %v", err)
	}
	instances := []Instance{
		{Name: "group-a", InstanceId: "ins-select-group-a", UserID: users[0].ID},
		{Name: "group-b", InstanceId: "ins-select-group-b", UserID: users[1].ID},
		{Name: "ungrouped", InstanceId: "ins-select-ungrouped", UserID: users[2].ID},
	}
	if err := db.Create(&instances).Error; err != nil {
		t.Fatalf("create instances: %v", err)
	}

	tests := []struct {
		name     string
		groupIDs []uint
		wantIDs  []uint
	}{
		{name: "one group", groupIDs: []uint{groupA.ID}, wantIDs: []uint{instances[0].ID}},
		{name: "ungrouped", groupIDs: []uint{0}, wantIDs: []uint{instances[2].ID}},
		{
			name:     "union with duplicate and ungrouped",
			groupIDs: []uint{0, groupA.ID, groupB.ID, groupA.ID},
			wantIDs:  []uint{instances[0].ID, instances[1].ID, instances[2].ID},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := db.Model(&Instance{}).Select("instances.id AS instance_id")
			query := FilterInstancesByUserGroups(ctx, base, tt.groupIDs)
			var targets []struct {
				InstanceID uint `gorm:"column:instance_id"`
			}
			if err := query.Order("instances.id ASC").Scan(&targets).Error; err != nil {
				t.Fatalf("FilterInstancesByUserGroups(%v) error = %v", tt.groupIDs, err)
			}
			seen := make(map[uint]int, len(targets))
			for _, target := range targets {
				seen[target.InstanceID]++
			}
			if len(seen) != len(tt.wantIDs) {
				t.Fatalf("FilterInstancesByUserGroups(%v) selected IDs = %v, want %v", tt.groupIDs, seen, tt.wantIDs)
			}
			for _, instanceID := range tt.wantIDs {
				if seen[instanceID] != 1 {
					t.Errorf("FilterInstancesByUserGroups(%v) selected instance %d %d times, want 1", tt.groupIDs, instanceID, seen[instanceID])
				}
			}
		})
	}
}
