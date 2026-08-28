package model

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"hatchery/common"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupPasswordlessLoginTokenTest(t *testing.T) (*gorm.DB, context.Context, context.Context) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	registerIdentifierCallbacks(db)
	if err := db.WithContext(common.WithSkipIdentifier(context.Background())).AutoMigrate(&PasswordlessLoginToken{}); err != nil {
		t.Fatalf("migrate token table: %v", err)
	}
	t.Cleanup(UseDBForTest(db))
	return db,
		common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-a"}),
		common.InjectTenant(context.Background(), common.TenantSnapshot{Identifier: "tenant-b"})
}

func TestPasswordlessLoginToken_CreateConsumeAndReplay(t *testing.T) {
	db, tenantA, _ := setupPasswordlessLoginTokenTest(t)
	now := time.Now().UTC()
	record, err := CreatePasswordlessLoginToken(tenantA, "digest-a", 42, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if record.Identifier != "tenant-a" || record.UserID != 42 {
		t.Fatalf("unexpected token record: %+v", record)
	}
	if _, err := CreatePasswordlessLoginToken(tenantA, "digest-a", 42, now.Add(2*time.Minute)); err == nil {
		t.Fatal("duplicate token digest should fail")
	}

	consumed, err := ConsumePasswordlessLoginToken(tenantA, "digest-a", now)
	if err != nil || consumed.ID != record.ID {
		t.Fatalf("consume token: record=%+v err=%v", consumed, err)
	}
	if err := db.WithContext(common.WithSkipIdentifier(context.Background())).First(&PasswordlessLoginToken{}, record.ID).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("consumed row still exists: %v", err)
	}
	if _, err := ConsumePasswordlessLoginToken(tenantA, "digest-a", now); !errors.Is(err, ErrPasswordlessLoginTokenInvalid) {
		t.Fatalf("replay error=%v", err)
	}
}

func TestPasswordlessLoginToken_ConcurrentSingleUse(t *testing.T) {
	_, tenantA, _ := setupPasswordlessLoginTokenTest(t)
	now := time.Now().UTC()
	if _, err := CreatePasswordlessLoginToken(tenantA, "digest-concurrent", 7, now.Add(time.Minute)); err != nil {
		t.Fatalf("create token: %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := ConsumePasswordlessLoginToken(tenantA, "digest-concurrent", now)
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	successes, invalid := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrPasswordlessLoginTokenInvalid):
			invalid++
		default:
			t.Fatalf("unexpected consume error: %v", err)
		}
	}
	if successes != 1 || invalid != 1 {
		t.Fatalf("successes=%d invalid=%d", successes, invalid)
	}
}

func TestPasswordlessLoginToken_ExpiryTenantIsolationAndCleanup(t *testing.T) {
	db, tenantA, tenantB := setupPasswordlessLoginTokenTest(t)
	now := time.Now().UTC()
	if _, err := CreatePasswordlessLoginToken(tenantA, "digest-a-expired", 1, now.Add(-time.Second)); err != nil {
		t.Fatalf("create expired tenant A token: %v", err)
	}
	if _, err := CreatePasswordlessLoginToken(tenantA, "digest-a-valid", 1, now.Add(time.Minute)); err != nil {
		t.Fatalf("create valid tenant A token: %v", err)
	}
	if _, err := CreatePasswordlessLoginToken(tenantB, "digest-b-expired", 2, now.Add(-time.Second)); err != nil {
		t.Fatalf("create expired tenant B token: %v", err)
	}

	if _, err := ConsumePasswordlessLoginToken(tenantA, "digest-a-expired", now); !errors.Is(err, ErrPasswordlessLoginTokenInvalid) {
		t.Fatalf("expired consume error=%v", err)
	}
	if _, err := ConsumePasswordlessLoginToken(tenantB, "digest-a-valid", now); !errors.Is(err, ErrPasswordlessLoginTokenInvalid) {
		t.Fatalf("cross-tenant consume error=%v", err)
	}
	if err := DeleteExpiredPasswordlessLoginTokens(tenantA, now); err != nil {
		t.Fatalf("cleanup tenant A: %v", err)
	}

	var records []PasswordlessLoginToken
	if err := db.WithContext(common.WithSkipIdentifier(context.Background())).Order("token_hash").Find(&records).Error; err != nil {
		t.Fatalf("list remaining records: %v", err)
	}
	if len(records) != 2 || records[0].TokenHash != "digest-a-valid" || records[1].TokenHash != "digest-b-expired" {
		t.Fatalf("unexpected remaining records: %+v", records)
	}
}

func TestPasswordlessLoginToken_DatabaseErrors(t *testing.T) {
	t.Run("query error", func(t *testing.T) {
		db, tenantA, _ := setupPasswordlessLoginTokenTest(t)
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatalf("get sql database: %v", err)
		}
		if err := sqlDB.Close(); err != nil {
			t.Fatalf("close sql database: %v", err)
		}
		if _, err := ConsumePasswordlessLoginToken(tenantA, "digest", time.Now().UTC()); err == nil || errors.Is(err, ErrPasswordlessLoginTokenInvalid) {
			t.Fatalf("expected database error, got %v", err)
		}
	})

	t.Run("delete error", func(t *testing.T) {
		db, tenantA, _ := setupPasswordlessLoginTokenTest(t)
		now := time.Now().UTC()
		if _, err := CreatePasswordlessLoginToken(tenantA, "digest-delete-error", 1, now.Add(time.Minute)); err != nil {
			t.Fatalf("create token: %v", err)
		}
		const callbackName = "test:passwordless_login_delete_error"
		if err := db.Callback().Delete().Before("gorm:delete").Register(callbackName, func(tx *gorm.DB) {
			if tx.Statement.Schema != nil && tx.Statement.Schema.Name == "PasswordlessLoginToken" {
				tx.AddError(errors.New("forced delete error"))
			}
		}); err != nil {
			t.Fatalf("register delete callback: %v", err)
		}
		t.Cleanup(func() {
			db.Callback().Delete().Remove(callbackName)
		})
		if _, err := ConsumePasswordlessLoginToken(tenantA, "digest-delete-error", now); err == nil || errors.Is(err, ErrPasswordlessLoginTokenInvalid) {
			t.Fatalf("expected delete error, got %v", err)
		}
	})
}
