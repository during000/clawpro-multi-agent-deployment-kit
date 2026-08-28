package model

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"hatchery/common"
	"hatchery/i18n"
)

// DistLock 基于 MySQL GET_LOCK() 的分布式锁。
// 锁名自动加上 identifier 前缀，实现多租户隔离。
// 每把锁持有一条独立的 *sql.Conn（advisory lock 绑定在连接上），
// 连接断开时锁自动释放，无孤锁问题。
//
// SQLite 模式下所有操作为空操作，业务代码无需判断数据库类型。
type DistLock struct {
	conn    *sql.Conn
	lockKey string
}

// lockName 生成带 identifier 前缀的锁名。
// 从 ctx 中的 TenantSnapshot 读取 identifier；若无或标记跳过则使用不带前缀的锁名（SQLite 或本地开发模式）。
// MySQL GET_LOCK 的 name 最大 64 字符。
func lockName(ctx context.Context, resource string) string {
	if common.ShouldSkipIdentifier(ctx) {
		return resource
	}
	if snap, ok := common.GetTenantSnapshot(ctx); ok && snap.Identifier != "" {
		return fmt.Sprintf("%s:%s", snap.Identifier, resource)
	}
	return resource
}

// AcquireLock 获取指定资源的分布式锁。
// resource: 锁标识（如 "instance:123"、"config"）。
// timeout: 等待获取锁的超时时间，0 表示不等待。
// 返回的 *DistLock 必须调用 Release() 释放；推荐配合 defer 使用。
//
// SQLite 模式下直接返回空壳 DistLock，不报错。
//
// 使用示例:
//
//	lock, err := model.AcquireLock(ctx, "instance:42", 5*time.Second)
//	if err != nil {
//	    return err
//	}
//	defer lock.Release()
//	// ... 持有锁期间执行临界区代码 ...
func AcquireLock(ctx context.Context, resource string, timeout time.Duration) (*DistLock, error) {
	// SQLite 模式：空操作
	if dbDriver != "mysql" {
		return &DistLock{}, nil
	}

	sqlDB, err := DB(ctx).DB()
	if err != nil {
		return nil, common.I18nRichError(err, i18n.MsgDistLockGetSQLDBFailed)
	}

	// 获取独立连接——advisory lock 与连接绑定
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return nil, common.I18nRichError(err, i18n.MsgDistLockAcquireConnFailed)
	}

	key := lockName(ctx, resource)
	if len(key) > 64 {
		conn.Close()
		return nil, common.I18nError(i18n.MsgDistLockNameTooLong, key, len(key))
	}
	timeoutSec := int(timeout.Seconds())

	var result sql.NullInt64
	err = conn.QueryRowContext(ctx, "SELECT GET_LOCK(?, ?)", key, timeoutSec).Scan(&result)
	if err != nil {
		conn.Close()
		return nil, common.I18nRichError(err, i18n.MsgDistLockGETLockFailed, key)
	}

	// GET_LOCK 返回值：1=成功, 0=超时, NULL=错误
	if !result.Valid {
		conn.Close()
		return nil, common.I18nError(i18n.MsgDistLockGETLockNullErr, key)
	}
	if result.Int64 != 1 {
		conn.Close()
		return nil, common.I18nError(i18n.MsgDistLockGETLockTimeout, key, timeout)
	}

	slog.Debug("[DistLock] 获取锁成功", "key", key)
	return &DistLock{conn: conn, lockKey: key}, nil
}

// TryLock 尝试立即获取锁，不等待。
// 成功返回 *DistLock，锁已被占用时返回 error。
// SQLite 模式下直接返回空壳 DistLock。
func TryLock(ctx context.Context, resource string) (*DistLock, error) {
	return AcquireLock(ctx, resource, 0)
}

// Release 释放锁并归还连接。
// 多次调用是安全的（幂等）。
func (l *DistLock) Release() {
	if l == nil || l.conn == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var result sql.NullInt64
	err := l.conn.QueryRowContext(ctx, "SELECT RELEASE_LOCK(?)", l.lockKey).Scan(&result)
	if err != nil {
		slog.Warn("[DistLock] 释放锁失败", "key", l.lockKey, "error", err)
	} else {
		slog.Debug("[DistLock] 释放锁成功", "key", l.lockKey)
	}

	l.conn.Close()
	l.conn = nil
}

// IsLockHeld 查询某个资源的锁是否被持有（任意连接）。
// 这是一个无副作用的检查，不获取也不释放锁。
// SQLite 模式下始终返回 false（未持有）。
func IsLockHeld(ctx context.Context, resource string) (bool, error) {
	if dbDriver != "mysql" {
		return false, nil
	}

	key := lockName(ctx, resource)
	if len(key) > 64 {
		return false, common.I18nError(i18n.MsgDistLockNameTooLong, key, len(key))
	}

	sqlDB, err := DB(ctx).DB()
	if err != nil {
		return false, common.I18nRichError(err, i18n.MsgDistLockGetSQLDBFailed)
	}
	var result sql.NullInt64
	err = sqlDB.QueryRowContext(ctx, "SELECT IS_FREE_LOCK(?)", key).Scan(&result)
	if err != nil {
		return false, common.I18nRichError(err, i18n.MsgDistLockISFreeLockFailed, key)
	}
	// IS_FREE_LOCK: 1=free, 0=held, NULL=error
	if !result.Valid {
		return false, common.I18nError(i18n.MsgDistLockISFreeLockNullErr, key)
	}
	return result.Int64 == 0, nil
}

// WithLock 在持有锁的期间执行 fn，执行完毕自动释放锁。
// 这是 AcquireLock + defer Release 的便捷封装。
func WithLock(ctx context.Context, resource string, timeout time.Duration, fn func(ctx context.Context) error) error {
	lock, err := AcquireLock(ctx, resource, timeout)
	if err != nil {
		return err
	}
	defer lock.Release()
	return fn(ctx)
}
