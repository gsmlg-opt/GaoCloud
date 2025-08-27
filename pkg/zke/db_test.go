package zke

import (
	"os"
	"testing"
	"time"

	ut "cement/unittest"
	"kvzoo"
	"kvzoo/backend/bolt"
	"zke/core"
	zketypes "zke/types"
)

func TestClusterDB(t *testing.T) {
	dbPath := "cluster.db"
	clusterName := "local"
	ut.WithTempFile(t, dbPath, func(t *testing.T, f *os.File) {

		db, err := bolt.New(f.Name())
		ut.Assert(t, err == nil, "create db should succeed: %s", err)

		tn, err := kvzoo.TableNameFromSegments(ZKEManagerDBTable)
		ut.Assert(t, err == nil, "get db table name should succeed: %s", err)

		table, err := db.CreateOrGetTable(tn)
		ut.Assert(t, err == nil, "create db table should succeed: %s", err)

		newClusterState := clusterState{
			FullState:  &core.FullState{},
			ZKEConfig:  &zketypes.ZKEConfig{},
			CreateTime: time.Now(),
			Created:    false,
			ScVersion:  "v1.0",
		}

		err = createOrUpdateClusterFromDB(clusterName, newClusterState, table)
		ut.Assert(t, err == nil, "create cluster from db should succeed: %s", err)

		newClusterState.Created = true
		err = createOrUpdateClusterFromDB(clusterName, newClusterState, table)
		ut.Assert(t, err == nil, "update cluster from db should succeed: %s", err)

		state, err := getClusterFromDB(clusterName, table)
		ut.Assert(t, err == nil, "get cluster from db should succeed: %s", err)
		ut.Assert(t, state.Created == newClusterState.Created, "after update cluster, it's created field should equal the value get from db")

		states, err := getClustersFromDB(table)
		ut.Assert(t, err == nil, "get clusters from db should succeed: %s", err)
		ut.Assert(t, len(states) == 1, "the clusters number that get from db should equal 1")

		err = deleteClusterFromDB(clusterName, table)
		ut.Assert(t, err == nil, "delete cluster from db should succeed: %s", err)

		state, err = getClusterFromDB(clusterName, table)
		ut.Assert(t, err == kvzoo.ErrNotFound, "get cluster from db after delete should get not found err: %s", err)
	})
}
