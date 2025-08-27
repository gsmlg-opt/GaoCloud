package tests

import (
	"sync"
	"testing"

	ut "cement/unittest"
	"kvzoo"
	"kvzoo/backend/bolt"
	"kvzoo/client"
	"kvzoo/server"
)

func mustChecksum(db kvzoo.DB) string {
	cs, err := db.Checksum()
	if err != nil {
		panic("db checksum get err:" + err.Error())
	}
	return cs
}

func TestBoltDBChecksum(t *testing.T) {
	db1, err := bolt.New("test1.db")
	ut.Assert(t, err == nil, "")
	db2, err := bolt.New("test2.db")
	ut.Assert(t, err == nil, "")
	defer func() {
		db1.Destroy()
		db2.Destroy()
	}()
	ut.Equal(t, mustChecksum(db1), mustChecksum(db2))

	tableName, _ := kvzoo.NewTableName("/xxxx/xx")
	keyPrefix, valuePrefix := "key", "v"
	keys, values := genData(keyPrefix, valuePrefix, 1000)
	loadDataToTableInParal(db1, tableName, keys, values)
	loadDataToTableInParal(db2, tableName, keys, values)
	ut.Equal(t, mustChecksum(db1), mustChecksum(db2))
}

func TestBoltDBTable(t *testing.T) {
	withBoltDB(t, testTable)
}

func withBoltDB(t *testing.T, test func(t *testing.T, db kvzoo.DB)) {
	db, err := bolt.New("test.db")
	ut.Equal(t, err, nil)
	defer db.Destroy()
	test(t, db)
}

func TestRemoteDBTable(t *testing.T) {
	withRemoteDB(t, testTable)
}

func withRemoteDB(t *testing.T, test func(t *testing.T, db kvzoo.DB)) {
	db1, err := bolt.New("s1.db")
	ut.Equal(t, err, nil)
	saddr1 := "127.0.0.1:7777"
	rdb1, err := server.New(saddr1, db1)
	ut.Equal(t, err, nil)
	go rdb1.Start()

	db2, err := bolt.New("s2.db")
	ut.Equal(t, err, nil)
	saddr2 := "127.0.0.1:7778"
	rdb2, err := server.New(saddr2, db2)
	ut.Equal(t, err, nil)
	go rdb2.Start()

	ldb, err := client.New(saddr1, []string{saddr2})
	ut.Equal(t, err, nil)
	_, err = ldb.Checksum()
	ut.Assert(t, err == nil, "")
	defer func() {
		ldb.Destroy()
		rdb1.Stop()
		rdb2.Stop()
	}()
	test(t, ldb)
}

func testTable(t *testing.T, db kvzoo.DB) {
	_, err := kvzoo.NewTableName("xxx")
	ut.Assert(t, err != nil, "")
	_, err = kvzoo.NewTableName("/xxx//")
	ut.Assert(t, err != nil, "")

	tn1, err := kvzoo.NewTableName("/xxx/good")
	ut.Assert(t, err == nil, "")
	tn2, err := kvzoo.NewTableName("/xxx/goodd")
	ut.Assert(t, err == nil, "")

	_, err = db.CreateOrGetTable(tn1)
	ut.Assert(t, err == nil, "")
	err = db.DeleteTable(tn2)
	ut.Assert(t, err != nil, "")
	err = db.DeleteTable(tn1)
	ut.Assert(t, err == nil, "")
}

func TestBoltDBAddAndGet(t *testing.T) {
	withBoltDB(t, testAddAndGet)
}

func TestRemoteDBAddAndGet(t *testing.T) {
	withRemoteDB(t, testAddAndGet)
}

func testAddAndGet(t *testing.T, db kvzoo.DB) {
	keyPrefix, valuePrefix := "key", "v"
	keys, values := genData(keyPrefix, valuePrefix, 1000)
	tableName, _ := kvzoo.NewTableName("/xxxx/xx")
	err := loadDataToTable(db, tableName, keys, values)
	ut.Equal(t, err, nil)
	ut.Assert(t, tableHasData(db, tableName, keys, values), "")
	err = db.DeleteTable(tableName)
	ut.Equal(t, err, nil)

	keys, values = genData(keyPrefix, valuePrefix, 10)
	err = loadDataToTableInParal(db, tableName, keys, values)
	ut.Equal(t, err, nil)
	ut.Assert(t, tableHasData(db, tableName, keys, values), "")
	ut.Assert(t, tableHasData(db, tableName, []string{"k1"}, []string{"v1"}) == false, "")
	db.DeleteTable(tableName)

	err = loadDataToTableInParal(db, tableName, []string{"k1", "k1"}, []string{"v1", "v2"})
	ut.Assert(t, err != nil, "")
	db.DeleteTable(tableName)

	err = loadDataToTable(db, tableName, []string{"k1", "k1"}, []string{"v1", "v2"})
	ut.Assert(t, err != nil, "")
	db.DeleteTable(tableName)
}

func TestBoltDBUpdate(t *testing.T) {
	withBoltDB(t, testUpdate)
}

func TestRemoteDBUpdate(t *testing.T) {
	withRemoteDB(t, testUpdate)
}

func testUpdate(t *testing.T, db kvzoo.DB) {
	keys, values := genData("key", "value", 1000)
	tableName, _ := kvzoo.NewTableName("/xxxx")
	err := loadDataToTable(db, tableName, keys, values)
	ut.Equal(t, err, nil)
	_, values = genData("key", "vv", 1000)
	err = updateDataInTable(db, tableName, keys, values)
	ut.Equal(t, err, nil)
	ut.Assert(t, tableHasData(db, tableName, keys, values), "")
	db.DeleteTable(tableName)

	keys, values = genData("k", "value", 10)
	err = loadDataToTable(db, tableName, keys, values)
	ut.Equal(t, err, nil)
	_, values = genData("k", "vvv", 10)
	err = updateDataInTableInParal(db, tableName, keys, values)
	ut.Equal(t, err, nil)
	ut.Assert(t, tableHasData(db, tableName, keys, values), "")

	err = updateDataInTableInParal(db, tableName, []string{"nk1", "nk2"}, []string{"v1", "v2"})
	ut.Assert(t, err != nil, "")

	err = updateDataInTable(db, tableName, []string{"key1", "key2"}, []string{"v1", "v2"})
	ut.Assert(t, err != nil, "")

	ut.Assert(t, tableHasData(db, tableName, keys, values), "")
}

func TestBoltDBDelete(t *testing.T) {
	withBoltDB(t, testDelete)
}

func TestRemoteDBDelete(t *testing.T) {
	withRemoteDB(t, testDelete)
}

func testDelete(t *testing.T, db kvzoo.DB) {
	keys, values := genData("key", "value", 1000)
	tableName, _ := kvzoo.NewTableName("/xxxx/xxx/xxxxx")
	err := loadDataToTable(db, tableName, keys, values)
	ut.Equal(t, err, nil)
	data, err := getTableData(db, tableName)
	ut.Equal(t, err, nil)
	ut.Equal(t, len(data), 1000)

	keys, values = genData("key", "value", 500)
	err = deleteDataInTable(db, tableName, keys, values)
	ut.Equal(t, err, nil)
	data, err = getTableData(db, tableName)
	ut.Equal(t, err, nil)
	ut.Equal(t, len(data), 500)
	db.DeleteTable(tableName)
	ut.Assert(t, tableDoesNotHasKeys(db, tableName, keys), "")

	keys, values = genData("key", "value", 100)
	err = loadDataToTable(db, tableName, keys, values)
	ut.Equal(t, err, nil)
	err = deleteDataInTable(db, tableName, []string{"k1", "k2"}, []string{"v1", "v2"})
	ut.Assert(t, err == nil, "")
	err = deleteDataInTableInParal(db, tableName, []string{"kk1", "kk2"}, []string{"v1", "v2"})
	ut.Assert(t, err == nil, "")
	err = deleteDataInTableInParal(db, tableName, keys, values)
	data, err = getTableData(db, tableName)
	ut.Equal(t, err, nil)
	ut.Equal(t, len(data), 0)
}

func TestBoltDBList(t *testing.T) {
	withBoltDB(t, testList)
}

func TestRemoteDBList(t *testing.T) {
	withRemoteDB(t, testList)
}

func testList(t *testing.T, db kvzoo.DB) {
	keys, values := genData("key", "value", 1000)
	tableName, _ := kvzoo.NewTableName("/xxxx/x/xxxxx")
	err := loadDataToTable(db, tableName, keys, values)
	ut.Equal(t, err, nil)
	data, err := getTableData(db, tableName)
	ut.Equal(t, err, nil)
	assertMapEqualsToSlices(t, data, keys, values)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			data, err := getTableData(db, tableName)
			ut.Equal(t, err, nil)
			assertMapEqualsToSlices(t, data, keys, values)
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestBoltDBNestedTable(t *testing.T) {
	withBoltDB(t, testNestedTable)
}

func TestRemoteDBNestedTable(t *testing.T) {
	withRemoteDB(t, testNestedTable)
}

func testNestedTable(t *testing.T, db kvzoo.DB) {
	t1, _ := kvzoo.NewTableName("/app/cd/ns1")
	keys, values := genData("key", "value", 1000)
	err := loadDataToTable(db, t1, keys, values)
	ut.Assert(t, err == nil, "")
	err = db.DeleteTable("/app/cd")
	ut.Assert(t, err == nil, "")
	data, err := getTableData(db, t1)
	ut.Equal(t, err, nil)
	ut.Equal(t, len(data), 0)

	loadDataToTable(db, t1, keys, values)
	data, _ = getTableData(db, t1)
	ut.Equal(t, len(data), 1000)

	t2, _ := kvzoo.NewTableName("/app/cd/ns2")
	loadDataToTable(db, t2, keys, values)
	data, _ = getTableData(db, t2)
	ut.Equal(t, len(data), 1000)

	tn, _ := kvzoo.NewTableName("/app")
	db.DeleteTable(tn)
	data, _ = getTableData(db, t1)
	ut.Equal(t, len(data), 0)
	data, _ = getTableData(db, t2)
	ut.Equal(t, len(data), 0)
}

func TestBoltDBTxRollback(t *testing.T) {
	withBoltDB(t, testTxRollback)
}

func TestRemoteDBTxRollback(t *testing.T) {
	withRemoteDB(t, testTxRollback)
}

func testTxRollback(t *testing.T, db kvzoo.DB) {
	tn, err := kvzoo.NewTableName("/good")
	ut.Assert(t, err == nil, "")

	table, err := db.CreateOrGetTable(tn)
	ut.Assert(t, err == nil, "")

	tx, err := table.Begin()
	ut.Assert(t, err == nil, "")
	keys, values := genData("k", "v", 10)
	for i := 0; i < 10; i++ {
		tx.Add(keys[i], []byte(values[i]))
	}
	tx.Rollback()
	data, err := getTableData(db, tn)
	ut.Assert(t, err == nil, "")
	ut.Equal(t, len(data), 0)

	loadDataToTable(db, tn, keys, values)
	tx, err = table.Begin()
	ut.Assert(t, err == nil, "")
	for _, key := range keys {
		tx.Delete(key)
	}
	tx.Rollback()
	ut.Assert(t, tableHasData(db, tn, keys, values), "")

	_, newValues := genData("k", "value", 10)
	tx, err = table.Begin()
	ut.Assert(t, err == nil, "")
	for i := 0; i < 10; i++ {
		tx.Update(keys[i], []byte(newValues[i]))
	}
	tx.Rollback()
	ut.Assert(t, tableHasData(db, tn, keys, values), "")
}
