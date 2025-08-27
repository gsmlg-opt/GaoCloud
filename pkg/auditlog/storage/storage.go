package storage

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"sync"

	"pkg/types"

	"kvzoo"
)

type StorageDriver interface {
	Add(a *types.AuditLog) error
	List() (types.AuditLogs, error)
}

type DefaultDriver struct {
	maxRecordCount int
	table          kvzoo.Table
	firstID        uint64
	currentID      uint64
	lock           sync.Mutex
}

func NewDefaultDriver(table kvzoo.Table, maxRecordCount int) (StorageDriver, error) {
	d := &DefaultDriver{
		maxRecordCount: maxRecordCount,
		table:          table,
	}

	if err := d.initDB(); err != nil {
		return d, err
	}
	return d, nil
}

func (d *DefaultDriver) initDB() error {
	logs, err := listFromDB(d.table)
	if err != nil {
		return err
	}

	logCount := len(logs)
	if logCount == 0 {
		return nil
	}

	sort.Sort(logs)

	d.firstID = logs[0].UID
	d.currentID = logs[logCount-1].UID
	return nil
}

func listFromDB(table kvzoo.Table) (types.AuditLogs, error) {
	tx, err := table.Begin()
	if err != nil {
		return nil, err
	}

	defer tx.Commit()
	values, err := tx.List()
	if err != nil {
		return nil, err
	}

	var logs types.AuditLogs
	for _, value := range values {
		var log types.AuditLog
		if err := json.Unmarshal(value, &log); err != nil {
			return nil, err
		}
		logs = append(logs, &log)
	}
	return logs, nil
}

func (d *DefaultDriver) Add(a *types.AuditLog) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.currentID-d.firstID > uint64(d.maxRecordCount-2) {
		if err := deleteFromDB(d.table, uintToStr(d.firstID)); err != nil {
			return err
		}
		d.firstID += 1
	}
	a.UID = d.currentID + 1
	a.SetID(uintToStr(d.currentID + 1))
	if err := addToDB(d.table, a); err != nil {
		return err
	}
	d.currentID += 1
	return nil
}

func addToDB(table kvzoo.Table, log *types.AuditLog) error {
	value, err := json.Marshal(log)
	if err != nil {
		return fmt.Errorf("marshal auditlog %s failed: %s", log.ID, err.Error())
	}

	tx, err := table.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction failed: %s", err.Error())
	}

	defer tx.Rollback()
	if err := tx.Add(log.ID, value); err != nil {
		return err
	}
	return tx.Commit()
}

func deleteFromDB(table kvzoo.Table, id string) error {
	tx, err := table.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()
	if err := tx.Delete(id); err != nil {
		return err
	}

	return tx.Commit()
}

func uintToStr(uid uint64) string {
	return strconv.FormatUint(uid, 10)
}

func (d *DefaultDriver) List() (types.AuditLogs, error) {
	return listFromDB(d.table)
}
