# 测试用数据库

## 初始化测试用 sqlite 数据库步骤

```go
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
if err != nil {
    t.Fatalf("打开 DB 失败: %v", err)
}

// 如果使用 :memory:，则设置最大连接数为 1
// 并发连接可能会导致 no such table 错误
sqlDB, _ := db.DB()
// 保证单个连接不会因为生命周期或者空闲而断开
sqlDB.SetConnMaxIdleTime(0)
sqlDB.SetConnMaxLifetime(0)
sqlDB.SetMaxOpenConns(1)

if err := db.AutoMigrate(
    &model.UserGroup{},
    &model.UserGroupMember{},
    &model.GroupClosure{},
    // ...
); err != nil {
    t.Fatalf("migrate 失败: %v", err)
}

// t *testing.T
// 测试用例结束后必须释放全局数据库句柄
t.Cleanup(model.UseDBForTest(db))
```

# 编写测试用例必须检查

## 异步任务
尽可能保证测试的代码不会创建异步任务（创建新的goroutine），即使创建了，也一定要保证测试用例结束时，该异步任务也被结束，例如测试以下函数时：

- HandleDistributeSkill 
- handleAdminDelModel
- handleSetModel

如果无法避免启动异步任务，则测试时将启动异步任务的函数 mock 掉，或者使用 WaitGroup 等待异步任务

- 使用 WaitGroup 参考 TestHandleDistributeSkill_MixedTypes_OnlyValidCreatesTask 用例
- 将异步任务发起者 mock 掉参考 TestHandleSetModel_CustomModel_E2E 用例，其 mock 了 syncHermesLLMToTDAI 函数

## 释放测试用数据库

最标准的方法是通过 go 单元测试框架自动清理机制：
```go
func TestTest(t *testing.T) {
    var db *gorm.DB
    // 初始化 db ...
    t.Cleanup(model.UseDBForTest(db))
}
```
如果需要中途更换数据库句柄，则：
```go
func TestTest(t *testing.T) {
    var db *gorm.DB
    // 初始化 db ...
    restore := model.UseDBForTest(db)
    // 释放前一个数据库句柄
    restore()
    // 也要记得测试用例结束后归还句柄
    t.Cleanup(model.UseNilDBForTest())

    // do testing
}
```
如果在测试后想要重新设置数据库，使用 SetDBForTest 而不是 UseDBForTest
```go
func TestTest(t *testing.T) {
    var db *gorm.DB
    // 初始化 db ...
    restore = model.UseDBForTest(db)

    t.Cleanup(func() {
        // 释放持有的句柄
        restore()
        // 设置全局句柄
        model.SetDBForTest(newDB)
    })
}
```

## 显式指定 dbDriver
有些函数可能使用了 model/distlock，其在释放锁的时候会主动销毁连接，如果使用 sqlite :memory:，连接被销毁了数据库也会被一并销毁，之后的连接会出现no such table的错误，这时候需要显式指定 dbDriver 为 sqlite:
```go
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
if err != nil {
    t.Fatalf("打开 DB 失败: %v", err)
}

// 如果使用 :memory:，则设置最大连接数为 1
// 并发连接可能会导致 no such table 错误
sqlDB, _ := db.DB()
sqlDB.SetConnMaxIdleTime(0)
sqlDB.SetConnMaxLifetime(0)
sqlDB.SetMaxOpenConns(1)

if err := db.AutoMigrate(
    &model.UserGroup{},
    &model.UserGroupMember{},
    &model.GroupClosure{},
    // ...
); err != nil {
    t.Fatalf("migrate 失败: %v", err)
}

// t *testing.T
// 测试用例结束后必须释放全局数据库句柄
// !!! 显式指定 sqlite
t.Cleanup(model.UseDBForTestWithDriver(db, "sqlite"))
```

# 覆盖率检查

## 增量覆盖率（推荐）

使用 CI 脚本检查 PR 的增量覆盖率（全量 + 增量）：

```bash
BASE_BRANCH=origin/master bash .ci/ci-check-coverage.sh
# 报告输出：coverage-report/index.html
```

默认增量覆盖率阈值 60%，可通过 `INCREMENTAL_THRESHOLD` 环境变量调整。

## 手动检查单个包

```bash
go test ./common/ -coverprofile=/tmp/cover.out -count=1
go tool cover -func=/tmp/cover.out | grep <函数名>
```
