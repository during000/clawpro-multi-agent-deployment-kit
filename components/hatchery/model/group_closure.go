package model

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"
)

// GroupClosure 分组闭包表，物化每个节点与其祖先的关系 + depth。
// 每个节点至少有一条自指行（ancestor_id = descendant_id，depth = 0）。
//
// 语义：若存在 (ancestor_id=A, descendant_id=D, depth=K)，则从 A 向下走 K 层可到达 D。
//
// 典型查询：
//   - 子树成员（含自身）：WHERE ancestor_id = X
//   - 祖先链（含自身）：WHERE descendant_id = X
//   - 是否 X 是 Y 的祖先：EXISTS (WHERE ancestor_id=X AND descendant_id=Y)
type GroupClosure struct {
	Identifier   string `gorm:"size:191;primaryKey;not null;default:''" json:"-"`
	AncestorID   uint   `gorm:"primaryKey;not null" json:"ancestor_id"`
	DescendantID uint   `gorm:"primaryKey;not null" json:"descendant_id"`
	Depth        int    `gorm:"not null;default:0" json:"depth"`
}

// TableName 指定表名（GORM 默认会将 GroupClosure 复数化为 group_closures，显式固定避免误差）。
func (GroupClosure) TableName() string { return "group_closure" }

// closureInsertForNewChildTx 新建分组时写 closure：
// 1) 自指行 (newID, newID, 0)
// 2) 继承 parent 的所有祖先：对每条 (a, parent, k)，插入 (a, newID, k+1)
//
// parentID=0 表示根组，只插入自指行。
// 调用方应保证同事务内 user_groups 行已插入（有 newID）。
//
// 实现：先 Find 父节点所有 ancestor 行到内存，再 +1 depth 后批量 Create —— 走
// GORM 回调，Query/Create 都会自动按 identifier 隔离，避免跨租户写入。
func closureInsertForNewChildTx(tx *gorm.DB, newID, parentID uint) error {
	// 自指（走 Create 回调，identifier 自动注入）
	self := GroupClosure{AncestorID: newID, DescendantID: newID, Depth: 0}
	if err := tx.Create(&self).Error; err != nil {
		return err
	}
	if parentID == 0 {
		return nil
	}
	// 取 parent 自身 + parent 的所有祖先（descendant_id = parentID）
	var parentAncestors []GroupClosure
	if err := tx.Where("descendant_id = ?", parentID).Find(&parentAncestors).Error; err != nil {
		return err
	}
	if len(parentAncestors) == 0 {
		return nil
	}
	rows := make([]GroupClosure, 0, len(parentAncestors))
	for _, a := range parentAncestors {
		rows = append(rows, GroupClosure{
			Identifier:   a.Identifier, // 已隔离过，与本租户 newID 一致
			AncestorID:   a.AncestorID,
			DescendantID: newID,
			Depth:        a.Depth + 1,
		})
	}
	return tx.CreateInBatches(rows, 200).Error
}

// closureMoveSubtreeTx 换父时调整 closure。
// 1) 删除被移动子树中每个节点与"旧祖先（不含子树内部）"的关联
// 2) 插入被移动子树中每个节点与"新祖先链"的关联
//
// rootID 是被移动子树根；newParentID 是新父（0 表示移动为根组）。
// 调用方应保证 user_groups 的 parent_id 已更新（或留到 caller 统一更新，均可，但 closure 调整顺序对正确性无影响）。
//
// 实现：完全走 GORM Find/Delete/Create 接口，identifier 自动隔离。
func closureMoveSubtreeTx(tx *gorm.DB, rootID, newParentID uint) error {
	// Step 1: 删除子树 × 旧祖先（子树外部）的关联
	// 1.1 取子树所有节点 id（含 root 自身）
	var subtreeRows []GroupClosure
	if err := tx.Where("ancestor_id = ?", rootID).Find(&subtreeRows).Error; err != nil {
		return err
	}
	if len(subtreeRows) == 0 {
		// 子树为空（root 自指行都没有），异常但不阻塞
		return nil
	}
	subtreeIDs := make([]uint, 0, len(subtreeRows))
	for _, r := range subtreeRows {
		subtreeIDs = append(subtreeIDs, r.DescendantID)
	}
	// 1.2 删除：descendant 在子树内 AND ancestor 在子树外
	if err := tx.Where("descendant_id IN ? AND ancestor_id NOT IN ?", subtreeIDs, subtreeIDs).
		Delete(&GroupClosure{}).Error; err != nil {
		return err
	}

	if newParentID == 0 {
		// 移动为根组：旧祖先已删除，无需插入新祖先
		return nil
	}

	// Step 2: 为子树每个节点与新父的每个祖先建立新关系
	// 取 newParent 自身 + 其祖先（sup.depth = newParent 到祖先 a 的距离）
	var supRows []GroupClosure
	if err := tx.Where("descendant_id = ?", newParentID).Find(&supRows).Error; err != nil {
		return err
	}
	if len(supRows) == 0 {
		return nil
	}
	// 笛卡尔积构造新行：对每对 (sup, sub) 插入 (sup.ancestor_id, sub.descendant_id, sup.depth + sub.depth + 1)
	rows := make([]GroupClosure, 0, len(supRows)*len(subtreeRows))
	for _, sup := range supRows {
		for _, sub := range subtreeRows {
			rows = append(rows, GroupClosure{
				Identifier:   sup.Identifier,
				AncestorID:   sup.AncestorID,
				DescendantID: sub.DescendantID,
				Depth:        sup.Depth + sub.Depth + 1,
			})
		}
	}
	return tx.CreateInBatches(rows, 500).Error
}

// closureDeleteNodeTx 删除分组时清理 closure。
// 语义：删除所有 closure 行 C 满足 C.ancestor_id = id OR C.descendant_id = id。
//
// 注意：只清除被删组与其它节点的关系；不递归清子组（子组应由调用方分别删除或事先检查阻塞）。
func closureDeleteNodeTx(tx *gorm.DB, id uint) error {
	return tx.Where("ancestor_id = ? OR descendant_id = ?", id, id).
		Delete(&GroupClosure{}).Error
}

// closureIsDescendantTx 判断 descendantID 是否在 ancestorID 的子树中（含自身）。
func closureIsDescendantTx(tx *gorm.DB, ancestorID, descendantID uint) (bool, error) {
	var count int64
	if err := tx.Model(&GroupClosure{}).
		Where("ancestor_id = ? AND descendant_id = ?", ancestorID, descendantID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// closureMaxRelativeDepthTx 查询子树内相对于 rootID 的最大深度（含自身）。
// 例如 root 下面有二代后代，返回 2。无后代返回 0。
func closureMaxRelativeDepthTx(tx *gorm.DB, rootID uint) (int, error) {
	var maxDepth int
	if err := tx.Model(&GroupClosure{}).
		Select("COALESCE(MAX(depth), 0)").
		Where("ancestor_id = ?", rootID).
		Scan(&maxDepth).Error; err != nil {
		return 0, err
	}
	return maxDepth, nil
}

// ClosureDescendants 查询某组及其所有后代的 ID 列表。
// includeSelf=true 时包含 rootID 自身；否则仅返回真子孙。
func ClosureDescendants(ctx context.Context, rootID uint, includeSelf bool) ([]uint, error) {
	return closureDescendantsDB(DB(ctx), rootID, includeSelf)
}

func closureDescendantsDB(db *gorm.DB, rootID uint, includeSelf bool) ([]uint, error) {
	q := db.Model(&GroupClosure{}).Where("ancestor_id = ?", rootID)
	if !includeSelf {
		q = q.Where("descendant_id <> ?", rootID)
	}
	var rows []GroupClosure
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, len(rows))
	for i, r := range rows {
		ids[i] = r.DescendantID
	}
	return ids, nil
}

// ClosureAncestors 查询某组的所有祖先 ID（从父到根，按 depth 升序从近到远）。
// includeSelf=true 时包含 id 自身。
func ClosureAncestors(ctx context.Context, id uint, includeSelf bool) ([]uint, error) {
	q := DB(ctx).Model(&GroupClosure{}).Where("descendant_id = ?", id)
	if !includeSelf {
		q = q.Where("ancestor_id <> ?", id)
	}
	var rows []GroupClosure
	if err := q.Order("depth ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	ids := make([]uint, len(rows))
	for i, r := range rows {
		ids[i] = r.AncestorID
	}
	return ids, nil
}

// ReconcileClosure 启动自检：对比 user_groups.parent_id 与 group_closure 一致性，
// 仅对差异行做最小化增量修复，不再 TRUNCATE 全表重建。
//
// 逻辑：
//  1. 按 user_groups 推导期望 closure 集合（key=(ancestor, descendant)，value=depth）
//  2. 读实际 closure 集合
//  3. 计算差集：
//     - missing：expected 中存在但 actual 缺失，或 depth 不一致 → 需要 INSERT
//     - extra  ：actual 中存在但 expected 没有，或 depth 不一致 → 需要 DELETE
//     - depth 不一致的同一 (a,d) 会同时出现在 missing 和 extra（旧行删、新行插）
//  4. 在单事务中先 DELETE 差异行、再批量 INSERT 缺失行
//
// 多租户：所有 DB 调用走 GORM Find/Delete/Create 接口，identifier 由全局回调
// 自动注入到 WHERE / 写入字段，因此 expected/actual 天然限定在当前租户范围，
// 差集计算和后续 DELETE/INSERT 均无需显式带 identifier。
//
// 优于旧实现的点：
//   - 不再因为一条脏数据而 DELETE 整张 group_closure（旧实现 risk：事务期间任何
//     并发查询会看到空表 / 写入失败丢历史）
//   - 完全一致时只是两次 SELECT，无写入
//
// 该函数不会抛错中断启动：失败只打 ERROR 日志。
func ReconcileClosure(ctx context.Context) {
	if gdb == nil {
		return
	}
	// Step 1: 读所有 user_groups（identifier 由回调自动过滤）
	var groups []UserGroup
	if err := DB(ctx).Select("id, parent_id").Find(&groups).Error; err != nil {
		slog.Error("ReconcileClosure: 读取 user_groups 失败", "error", err)
		return
	}
	byID := make(map[uint]UserGroup, len(groups))
	for _, g := range groups {
		byID[g.ID] = g
	}

	type closureKey struct {
		AncestorID   uint
		DescendantID uint
	}
	expected := make(map[closureKey]int, len(groups)*2) // key → depth
	for _, g := range groups {
		// 自指
		expected[closureKey{g.ID, g.ID}] = 0
		// 向上找祖先
		cur := g.ParentID
		depth := 1
		guard := MaxGroupDepth + 5
		for cur != 0 && guard > 0 {
			p, ok := byID[cur]
			if !ok {
				break
			}
			expected[closureKey{p.ID, g.ID}] = depth
			cur = p.ParentID
			depth++
			guard--
		}
	}

	// Step 2: 读 closure 实际（identifier 由回调自动过滤）
	var actual []GroupClosure
	if err := DB(ctx).Find(&actual).Error; err != nil {
		slog.Error("ReconcileClosure: 读取 group_closure 失败", "error", err)
		return
	}
	actualMap := make(map[closureKey]int, len(actual))
	for _, a := range actual {
		actualMap[closureKey{a.AncestorID, a.DescendantID}] = a.Depth
	}

	// Step 3: 计算差集
	var toInsert []GroupClosure
	for k, expDepth := range expected {
		actDepth, ok := actualMap[k]
		if !ok || actDepth != expDepth {
			toInsert = append(toInsert, GroupClosure{
				AncestorID:   k.AncestorID,
				DescendantID: k.DescendantID,
				Depth:        expDepth,
			})
		}
	}
	var toDelete []closureKey
	for k, actDepth := range actualMap {
		expDepth, ok := expected[k]
		if !ok || actDepth != expDepth {
			toDelete = append(toDelete, k)
		}
	}

	if len(toInsert) == 0 && len(toDelete) == 0 {
		return // 完全一致
	}

	slog.Warn("ReconcileClosure: user_groups 与 group_closure 不一致，触发增量修复",
		"to_insert", len(toInsert), "to_delete", len(toDelete),
		"expected_rows", len(expected), "actual_rows", len(actual))

	// Step 4: 事务内先删差异行、再插缺失行（identifier 由回调注入）
	start := time.Now()
	err := DB(ctx).Transaction(func(tx *gorm.DB) error {
		for _, k := range toDelete {
			if err := tx.Where("ancestor_id = ? AND descendant_id = ?",
				k.AncestorID, k.DescendantID).
				Delete(&GroupClosure{}).Error; err != nil {
				return err
			}
		}
		if len(toInsert) > 0 {
			if err := tx.CreateInBatches(toInsert, 500).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		slog.Error("ReconcileClosure: 修复 group_closure 失败", "error", err)
		return
	}
	slog.Info("ReconcileClosure: group_closure 增量修复完成",
		"inserted", len(toInsert), "deleted", len(toDelete), "elapsed", time.Since(start))
}

// ErrClosureInconsistent 指示 closure 数据自检不一致（当前不抛；保留供将来严格模式）。
var ErrClosureInconsistent = errors.New("group_closure inconsistent")

// 下列函数占位以便 linter 识别（无实际用途，避免 unused 警告）。
var _ = fmt.Sprint
