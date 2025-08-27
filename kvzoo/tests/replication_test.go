package tests

import (
	"fmt"
	"testing"

	ut "cement/unittest"
	"kvzoo"
	"kvzoo/backend/bolt"
	"kvzoo/client"
	"kvzoo/server"
)

type testEnv struct {
	backends []kvzoo.DB
	servers  []*server.KVGRPCServer
	proxy    kvzoo.DB
}

func newTestEnv(t *testing.T, count int) *testEnv {
	var backends []kvzoo.DB
	var servers []*server.KVGRPCServer
	var addrs []string
	startPort := 7700

	for i := 0; i < count; i++ {
		db, err := bolt.New(fmt.Sprintf("s%d.db", i))
		ut.Equal(t, err, nil)
		addr := fmt.Sprintf("127.0.0.1:%d", startPort+i)
		addrs = append(addrs, addr)
		rdb, err := server.New(addr, db)
		ut.Equal(t, err, nil)
		go rdb.Start()
		backends = append(backends, db)
		servers = append(servers, rdb)
	}

	proxy, err := client.New(addrs[0], addrs[1:])
	ut.Equal(t, err, nil)

	return &testEnv{
		backends: backends,
		servers:  servers,
		proxy:    proxy,
	}
}

func (e *testEnv) clean() {
	e.proxy.Destroy()
	for _, s := range e.servers {
		s.Stop()
	}
}

func (e *testEnv) checkTableHasData(t *testing.T, tableName kvzoo.TableName, keys, values []string) {
	for _, db := range e.backends {
		ut.Assert(t, tableHasData(db, tableName, keys, values), "")
	}
}

func (e *testEnv) checkTableIsEmpty(t *testing.T, tableName kvzoo.TableName) {
	for _, db := range e.backends {
		data, err := getTableData(db, tableName)
		ut.Equal(t, err, nil)
		ut.Equal(t, len(data), 0)
	}
}

func TestDBReplication(t *testing.T) {
	e := newTestEnv(t, 5)
	defer e.clean()

	//replication after add
	keyCount := 1000
	keyPrefix, valuePrefix := "key", "value"
	keys, values := genData(keyPrefix, valuePrefix, keyCount)
	tableName1, _ := kvzoo.NewTableName("/xxxx/xx/xxdd")
	err := loadDataToTable(e.proxy, tableName1, keys, values)
	ut.Equal(t, err, nil)
	tableName2, _ := kvzoo.NewTableName("/xxxxyxxxx")
	err = loadDataToTable(e.proxy, tableName2, keys, values)
	ut.Equal(t, err, nil)
	e.checkTableHasData(t, tableName1, keys, values)
	e.checkTableHasData(t, tableName2, keys, values)
	_, err = e.proxy.Checksum()
	ut.Assert(t, err == nil, "")
	err = e.proxy.DeleteTable(tableName1)
	err = e.proxy.DeleteTable(tableName2)
	ut.Equal(t, err, nil)

	keyPrefix, valuePrefix = "k", "v"
	keys, values = genData(keyPrefix, valuePrefix, keyCount)
	tableName, _ := kvzoo.NewTableName("/abcxx")
	err = loadDataToTableInParal(e.proxy, tableName, keys, values)
	ut.Equal(t, err, nil)

	e.checkTableHasData(t, tableName, keys, values)
	_, err = e.proxy.Checksum()
	ut.Assert(t, err == nil, "")

	//replication after update
	keyPrefix, valuePrefix = "k", "vvv"
	keys, values = genData(keyPrefix, valuePrefix, keyCount)
	tableName, _ = kvzoo.NewTableName("/abcxx")
	err = updateDataInTableInParal(e.proxy, tableName, keys, values)
	ut.Equal(t, err, nil)
	e.checkTableHasData(t, tableName, keys, values)

	_, err = e.proxy.Checksum()
	ut.Assert(t, err == nil, "")

	err = deleteDataInTableInParal(e.proxy, tableName, keys, values)
	ut.Equal(t, err, nil)
	e.checkTableIsEmpty(t, tableName)

	_, err = e.proxy.Checksum()
	ut.Assert(t, err == nil, "")
}
