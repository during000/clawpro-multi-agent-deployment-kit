package task

import (
	"context"
	"testing"
	"time"

	"hatchery/model"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestInstanceAdjustmentPoller_RegistrationContract(t *testing.T) {
	var found *TaskDef
	for i := range taskRegistry {
		if taskRegistry[i].Name == "instance-adjustment-poller" {
			copyValue := taskRegistry[i]
			found = &copyValue
			break
		}
	}
	if found == nil {
		t.Fatal("instance-adjustment-poller is not registered")
	}
	if found.Interval != 5*time.Second || found.InitialDelay != 5*time.Second || !found.NeedDistLock || !found.PerTenant || found.RunFunc == nil {
		t.Fatalf("poller definition=%+v", *found)
	}
}

func TestInstanceAdjustmentPoller_NoQueuedWorkReturns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.Instance{}); err != nil {
		t.Fatalf("migrate Instance: %v", err)
	}
	t.Cleanup(model.UseDBForTestWithDriver(db, "sqlite"))
	runInstanceAdjustmentPoller(context.Background())
}
