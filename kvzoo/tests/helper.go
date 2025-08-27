package tests

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"

	"cement/log"
	ut "cement/unittest"
	"kvzoo"
)

func init() {
	log.InitLogger(log.Debug)
}

func genData(keyPrefix, valuePrefix string, count int) ([]string, []string) {
	keys := make([]string, 0, count)
	values := make([]string, 0, count)
	for i := 0; i < count; i++ {
		keys = append(keys, fmt.Sprintf("%s%d", keyPrefix, i))
		values = append(values, fmt.Sprintf("%s%d", valuePrefix, i))
	}
	return keys, values
}

func loadDataToTable(db kvzoo.DB, tableName kvzoo.TableName, keys, values []string) error {
	return applyToTable(db, tableName, func(tx kvzoo.Transaction) dbOp {
		return tx.Add
	}, keys, values)
}

func loadDataToTableInParal(db kvzoo.DB, tableName kvzoo.TableName, keys, values []string) error {
	return applyToTableInParal(db, tableName, func(tx kvzoo.Transaction) dbOp {
		return tx.Add
	}, keys, values)
}

type dbOpGen func(kvzoo.Transaction) dbOp
type dbOp func(string, []byte) error

func applyToTable(db kvzoo.DB, tableName kvzoo.TableName, opGen dbOpGen, keys, values []string) error {
	table, err := db.CreateOrGetTable(tableName)
	if err != nil {
		return err
	}

	tx, err := table.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	op := opGen(tx)
	for i := 0; i < len(keys); i++ {
		if err := op(keys[i], []byte(values[i])); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func applyToTableInParal(db kvzoo.DB, tableName kvzoo.TableName, opGen dbOpGen, keys, values []string) error {
	table, err := db.CreateOrGetTable(tableName)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(keys))
	for i := 0; i < len(keys); i++ {
		wg.Add(1)
		go func(k, v string) {
			defer wg.Done()
			tx, err := table.Begin()
			if err != nil {
				errors <- err
				return
			}
			op := opGen(tx)
			if err := op(k, []byte(v)); err != nil {
				errors <- err
				tx.Rollback()
				return
			}
			if err := tx.Commit(); err != nil {
				errors <- err
				return
			}
		}(keys[i], values[i])
	}
	wg.Wait()
	if len(errors) != 0 {
		return <-errors
	} else {
		return nil
	}
}

func tableHasData(db kvzoo.DB, tableName kvzoo.TableName, keys, values []string) bool {
	table, err := db.CreateOrGetTable(tableName)
	if err != nil {
		return false
	}

	tx, err := table.Begin()
	if err != nil {
		return false
	}
	defer tx.Rollback()

	for i := 0; i < len(keys); i++ {
		value, err := tx.Get(keys[i])
		if err != nil {
			return false
		}

		if string(value) != values[i] {
			return false
		}
	}

	return true
}

func tableDoesNotHasKeys(db kvzoo.DB, tableName kvzoo.TableName, keys []string) bool {
	table, err := db.CreateOrGetTable(tableName)
	if err != nil {
		return false
	}

	tx, err := table.Begin()
	if err != nil {
		return false
	}
	defer tx.Rollback()

	for i := 0; i < len(keys); i++ {
		_, err := tx.Get(keys[i])
		if err != kvzoo.ErrNotFound {
			return false
		}

	}

	return true
}

func updateDataInTable(db kvzoo.DB, tableName kvzoo.TableName, keys, values []string) error {
	return applyToTable(db, tableName, func(tx kvzoo.Transaction) dbOp {
		return tx.Update
	}, keys, values)
}

func updateDataInTableInParal(db kvzoo.DB, tableName kvzoo.TableName, keys, values []string) error {
	return applyToTableInParal(db, tableName, func(tx kvzoo.Transaction) dbOp {
		return tx.Update
	}, keys, values)
}

func deleteDataInTable(db kvzoo.DB, tableName kvzoo.TableName, keys, values []string) error {
	return applyToTable(db, tableName, func(tx kvzoo.Transaction) dbOp {
		return func(k string, v []byte) error {
			return tx.Delete(k)
		}
	}, keys, values)
}

func deleteDataInTableInParal(db kvzoo.DB, tableName kvzoo.TableName, keys, values []string) error {
	return applyToTableInParal(db, tableName, func(tx kvzoo.Transaction) dbOp {
		return func(k string, v []byte) error {
			return tx.Delete(k)
		}
	}, keys, values)
}

func getTableData(db kvzoo.DB, tableName kvzoo.TableName) (map[string][]byte, error) {
	table, err := db.CreateOrGetTable(tableName)
	if err != nil {
		return nil, err
	}

	tx, err := table.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	return tx.List()
}

func assertMapEqualsToSlices(t *testing.T, m map[string][]byte, keys, values []string) {
	ut.Equal(t, len(m), len(keys))
	for i := 0; i < len(keys); i++ {
		ut.Equal(t, string(m[keys[i]]), values[i])
	}
}

func md5OfFile(filePath string) string {
	f, err := os.Open(filePath)
	if err != nil {
		panic(fmt.Sprintf("open %s failed:%v", filePath, err.Error()))
	}
	defer f.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, f); err != nil {
		panic(fmt.Sprintf("copy %s failed:%v", filePath, err.Error()))
	}

	hashInBytes := hash.Sum(nil)[:16]
	return hex.EncodeToString(hashInBytes)
}
