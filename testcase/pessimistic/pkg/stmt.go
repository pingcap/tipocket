package pkg

import (
	"context"
	"fmt"
	"math/rand"
	"strings"

	"github.com/cznic/mathutil"
	_ "github.com/go-sql-driver/mysql" // mysql
)

func (se *Session) plainSelect(ctx context.Context) error {
	sql := fmt.Sprintf("select * from %s%d where id between ? and ?", tableNamePrefix, 0)
	beginRowID := se.ran.nextRowID()
	endRowID := beginRowID + 10
	return se.executeSelect(ctx, sql, beginRowID, endRowID)
}

func (se *Session) selectForUpdate(ctx context.Context) error {
	sql := fmt.Sprintf("select * from %s%d where id in (?, ?) for update", tableNamePrefix, 0)
	return se.executeSelect(ctx, sql, se.ran.nextRowID(), se.ran.nextRowID())
}

func (se *Session) updateMultiTableSimple(ctx context.Context) error {
	fromTableID, toTableID, fromRowID, toRowID := se.ran.nextTableAndRowPairs()
	err := se.executeDML(
		ctx,
		fmt.Sprintf("update %s%d set c = c - 1 where id = ?", tableNamePrefix, fromTableID),
		fromRowID)
	if err != nil {
		return err
	}
	return se.executeDML(
		ctx,
		fmt.Sprintf("update %s%d set c = c + 1 where id = ?", tableNamePrefix, toTableID),
		toRowID)
}

func (se *Session) updateMultiTableIndex(ctx context.Context) error {
	fromTableID, toTableID, fromRowID, toRowID := se.ran.nextTableAndRowPairs()
	err := se.executeDML(
		ctx,
		fmt.Sprintf("update %s%d set i = i + 1, c = c - 1 where id = ?", tableNamePrefix, fromTableID),
		fromRowID)
	if err != nil {
		return err
	}
	return se.executeDML(
		ctx,
		fmt.Sprintf("update %s%d set i = i + 1, c = c + 1 where id = ?", tableNamePrefix, toTableID),
		toRowID)
}

// updateUniqueIndex make sure there is no conflict on the unique index by randomly generate the unique index value
func (se *Session) updateMultiTableUniqueIndex(ctx context.Context) error {
	sql := fmt.Sprintf("update %s%d set u = ?, c = c where id = ?", tableNamePrefix, se.ran.nextTableID())
	return se.executeDML(ctx, sql, se.ran.nextUniqueIndex(), se.ran.nextRowID())
}

func (se *Session) updateMultiTableRange(ctx context.Context) error {
	fromTableID, toTableID, fromRowID, toRowID := se.ran.nextTableAndRowPairs()
	beginRowID := mathutil.MinUint64(fromRowID, toRowID) + 2
	err := se.executeDML(
		ctx,
		fmt.Sprintf("update %s%d set c = c - 1 where id between ? and ?", tableNamePrefix, fromTableID),
		beginRowID,
		beginRowID+8)
	if err != nil {
		return err
	}
	return se.executeDML(
		ctx,
		fmt.Sprintf("update %s%d set c = c + 1 where id between ? and ?", tableNamePrefix, toTableID),
		beginRowID,
		beginRowID+8)
}

func (se *Session) updateMultiTableIndexRange(ctx context.Context) error {
	fromTableID, toTableID, fromRowID, toRowID := se.ran.nextTableAndRowPairs()
	beginRowID := mathutil.MinUint64(fromRowID, toRowID) + 2
	err := se.executeDML(
		ctx,
		fmt.Sprintf("update %s%d set i = i + 1, c = c - 1 where id between ? and ?", tableNamePrefix, fromTableID),
		beginRowID,
		beginRowID+8)
	if err != nil {
		return err
	}
	return se.executeDML(
		ctx,
		fmt.Sprintf("update %s%d set i = i + 1, c = c + 1 where id between ? and ?", tableNamePrefix, toTableID),
		beginRowID,
		beginRowID+8)
}

func (se *Session) replace(ctx context.Context) error {
	var fromCnt, toCnt int64
	fromTableID, toTableID, fromRowID, toRowID := se.ran.nextTableAndRowPairs()
	sql := fmt.Sprintf("select c from %s%d where id = ? for update", tableNamePrefix, fromTableID)
	row := se.conn.QueryRowContext(ctx, sql, fromRowID)
	err := row.Scan(&fromCnt)
	if err != nil {
		return err
	}
	sql = fmt.Sprintf("select c from %s%d where id = ? for update", tableNamePrefix, toTableID)
	row = se.conn.QueryRowContext(ctx, sql, toRowID)
	err = row.Scan(&toCnt)
	if err != nil {
		return err
	}
	sql = fmt.Sprintf("replace into %s%d values (?, ?, ?, ?, ?)", tableNamePrefix, fromTableID)
	if err = se.executeDML(ctx, sql, fromRowID, se.ran.nextUniqueIndex(), fromRowID, toCnt, randExceededLargeEntry()); err != nil {
		return err
	}
	sql = fmt.Sprintf("replace into %s%d values (?, ?, ?, ?, ?)", tableNamePrefix, toTableID)
	return se.executeDML(ctx, sql, toRowID, se.ran.nextUniqueIndex(), toRowID, fromCnt, randExceededLargeEntry())
}

func (se *Session) insertOnDuplicateUpdate(ctx context.Context) error {
	var fromCnt, toCnt int64
	fromTableID, toTableID, fromRowID, toRowID := se.ran.nextTableAndRowPairs()
	sql := fmt.Sprintf("select c from %s%d where id = ? for update", tableNamePrefix, fromTableID)
	row := se.conn.QueryRowContext(ctx, sql, fromRowID)
	err := row.Scan(&fromCnt)
	if err != nil {
		return err
	}
	sql = fmt.Sprintf("select c from %s%d where id = ? for update", tableNamePrefix, toTableID)
	row = se.conn.QueryRowContext(ctx, sql, toRowID)
	err = row.Scan(&toCnt)
	if err != nil {
		return err
	}
	sql = fmt.Sprintf("insert into %s%d values (?, ?, ?, ?, ?) on duplicate key update id=?, u=?, i=?, c=?", tableNamePrefix, fromTableID)
	if err = se.executeDML(ctx, sql, fromRowID, se.ran.nextUniqueIndex(), fromRowID, toCnt, randExceededLargeEntry(), fromRowID, se.ran.nextUniqueIndex(), fromRowID, toCnt); err != nil {
		return err
	}
	sql = fmt.Sprintf("insert into %s%d values (?, ?, ?, ?, ?) on duplicate key update id=?, u=?, i=?, c=?", tableNamePrefix, toTableID)
	return se.executeDML(ctx, sql, toRowID, se.ran.nextUniqueIndex(), toRowID, fromCnt, randExceededLargeEntry(), toRowID, se.ran.nextUniqueIndex(), toRowID, fromCnt)
}

func (se *Session) deleteInsert(ctx context.Context) error {
	var cntFrom, cntTo int64
	fromTableID, toTableID, fromRowID, toRowID := se.ran.nextTableAndRowPairs()
	sql := fmt.Sprintf("select c from %s%d where id = ? for update", tableNamePrefix, fromTableID)
	row := se.conn.QueryRowContext(ctx, sql, fromRowID)
	err := row.Scan(&cntFrom)
	if err != nil {
		return err
	}
	sql = fmt.Sprintf("delete from %s%d where id = ?", tableNamePrefix, fromTableID)
	if err = se.executeDML(ctx, sql, fromRowID); err != nil {
		return err
	}

	sql = fmt.Sprintf("select c from %s%d where id = ? for update", tableNamePrefix, toTableID)
	row = se.conn.QueryRowContext(ctx, sql, fromRowID)
	err = row.Scan(&cntTo)
	if err != nil {
		return err
	}
	sql = fmt.Sprintf("delete from %s%d where id = ?", tableNamePrefix, toTableID)
	if err = se.executeDML(ctx, sql, toRowID); err != nil {
		return err
	}

	sql = fmt.Sprintf("insert into %s%d values (?, ?, ?, ?, ?)", tableNamePrefix, fromTableID)
	if err = se.executeDML(ctx, sql, fromRowID, se.ran.nextUniqueIndex(), fromRowID, cntTo, randExceededLargeEntry()); err != nil {
		return err
	}
	sql = fmt.Sprintf("insert into %s%d values (?, ?, ?, ?, ?)", tableNamePrefix, toTableID)
	return se.executeDML(ctx, sql, toRowID, se.ran.nextUniqueIndex(), toRowID, cntFrom, randExceededLargeEntry())
}

func (se *Session) insertThenDelete(ctx context.Context) error {
	tableID := se.ran.nextTableID()
	rowID := se.ran.nextNonExistRowID()
	sql := fmt.Sprintf("insert into %s%d values (?, ?, ?, ?, ?)", tableNamePrefix, tableID)
	if err := se.executeDML(ctx, sql, rowID, se.ran.nextUniqueIndex(), rowID, 0, randExceededLargeEntry()); err != nil {
		return err
	}
	sql = fmt.Sprintf("delete from %s%d where id = ?", tableNamePrefix, tableID)
	return se.executeDML(ctx, sql, rowID)
}

func (se *Session) insertIgnore(ctx context.Context) error {
	fromTableID, toTableID, fromRowID, toRowID := se.ran.nextTableAndRowPairs()
	var cntFrom, cntTo int64

	sql := fmt.Sprintf("insert ignore into %s%d values (?, ?, ?, ?, ?)", tableNamePrefix, fromTableID)
	if err := se.executeDML(ctx, sql, fromRowID, 0, fromRowID, 0, randExceededLargeEntry()); err != nil {
		return err
	}
	sql = fmt.Sprintf("insert ignore into %s%d values (?, ?, ?, ?, ?)", tableNamePrefix, toTableID)
	if err := se.executeDML(ctx, sql, toRowID, 0, toRowID, 0, randExceededLargeEntry()); err != nil {
		return err
	}
	sql = fmt.Sprintf("select c from %s%d where id = ? for update", tableNamePrefix, fromTableID)
	row := se.conn.QueryRowContext(ctx, sql, fromRowID)
	err := row.Scan(&cntFrom)
	if err != nil {
		return err
	}
	sql = fmt.Sprintf("select c from %s%d where id = ? for update", tableNamePrefix, toTableID)
	row = se.conn.QueryRowContext(ctx, sql, toRowID)
	err = row.Scan(&cntTo)
	if err != nil {
		return err
	}
	sql = fmt.Sprintf("insert into %s%d values (?, ?, ?, ?, ?) on duplicate key update c=c+1", tableNamePrefix, fromTableID)
	if err := se.executeDML(ctx, sql, fromRowID, 0, fromRowID, 0, randExceededLargeEntry()); err != nil {
		return err
	}
	sql = fmt.Sprintf("insert into %s%d values (?, ?, ?, ?, ?) on duplicate key update c=c-1", tableNamePrefix, toTableID)
	return se.executeDML(ctx, sql, toRowID, 0, toRowID, 0, randExceededLargeEntry())
}

func (se *Session) insertIntoSelect(ctx context.Context) error {
	var cntFrom, cntTo int64
	fromTableID := se.ran.nextTableID()
	toTableID := se.ran.nextTableID()
	rowID := se.ran.nextRowID()

	sql := fmt.Sprintf("select c from %s%d where id = ? for update", tableNamePrefix, fromTableID)
	row := se.conn.QueryRowContext(ctx, sql, rowID)
	err := row.Scan(&cntFrom)
	if err != nil {
		return err
	}

	sql = fmt.Sprintf("select c from %s%d where id = ? for update", tableNamePrefix, toTableID)
	row = se.conn.QueryRowContext(ctx, sql, rowID)
	err = row.Scan(&cntTo)
	if err != nil {
		return err
	}

	sql = fmt.Sprintf("delete from %s%d where id = ?", tableNamePrefix, toTableID)
	if err = se.executeDML(ctx, sql, rowID); err != nil {
		return err
	}

	sql = fmt.Sprintf("insert into %s%d select * from %s%d where id=?", tableNamePrefix, toTableID, tableNamePrefix, fromTableID)
	if err = se.executeDML(ctx, sql, rowID); err != nil {
		return err
	}

	if cntTo-cntFrom == 0 {
		return nil
	}
	sql = fmt.Sprintf("update %s%d set c = c + ? where id=?", tableNamePrefix, toTableID)
	return se.executeDML(ctx, sql, cntTo-cntFrom, rowID)
}

func (se *Session) randSizeExceededTransaciton(ctx context.Context) error {
	if rand.Intn(50) != 0 {
		return nil
	}
	size := 1 * 1024 * 1024
	var b strings.Builder
	b.Grow(size)
	for i := 0; i < size; i++ {
		b.WriteString("0")
	}
	m := b.String()
	tableID := se.ran.nextTableID()
	var err error
	for i := 0; i < 100; i++ {
		rowID := se.ran.nextNonExistRowID()
		// Transaction is expected to fail
		sql := fmt.Sprintf("insert into %s%d values(?, ?, ?, ?, ?)", tableNamePrefix, tableID)
		if err = se.executeDML(ctx, sql, rowID, rowID, rowID, i, m); err != nil {
			return err
		}
	}
	return nil
}

func randExceededLargeEntry() string {
	if rand.Intn(50) == 0 {
		size := 6 * 1024 * 1024
		var b strings.Builder
		b.Grow(size)
		for i := 0; i < size; i++ {
			b.WriteString("1")
		}
		return b.String()
	}
	return ""
}
