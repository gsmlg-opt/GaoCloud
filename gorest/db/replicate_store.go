package db

import (
	"gorest/resource"
)

type ReplicateStore struct {
	master *RStore
	slave  *RStore
}

type ReplicateTx struct {
	masterTx Transaction
	slaveTx  Transaction
}

var _ ResourceStore = &ReplicateStore{}

func NewReplicateStore(mastConnStr, slaveConnStr string, meta *ResourceMeta) (*ReplicateStore, error) {
	master, err := NewRStore(mastConnStr, meta)
	if err != nil {
		return nil, err
	}

	slave, err := NewRStore(slaveConnStr, meta)
	if err != nil {
		return nil, err
	}

	return &ReplicateStore{
		master: master,
		slave:  slave,
	}, nil
}

func (s *ReplicateStore) Close() {
	s.master.Close()
	s.slave.Close()
}

func (s *ReplicateStore) Clean() {
	s.master.Clean()
	s.slave.Clean()
}

func (s *ReplicateStore) Begin() (Transaction, error) {
	masterTx, err := s.master.Begin()
	if err != nil {
		return nil, err
	}

	slaveTx, err := s.slave.Begin()
	if err != nil {
		masterTx.Rollback()
		return nil, err
	}

	return ReplicateTx{
		masterTx: masterTx,
		slaveTx:  slaveTx,
	}, nil
}

func (tx ReplicateTx) Insert(r resource.Resource) (resource.Resource, error) {
	nr, err := tx.masterTx.Insert(r)
	if err != nil {
		return nil, err
	}

	return tx.slaveTx.Insert(nr)
}

func (tx ReplicateTx) GetOwned(owner ResourceType, ownerID string, owned ResourceType) (interface{}, error) {
	return tx.masterTx.GetOwned(owner, ownerID, owned)
}

func (tx ReplicateTx) FillOwned(owner ResourceType, ownerID string, out interface{}) error {
	return tx.masterTx.FillOwned(owner, ownerID, out)
}

func (tx ReplicateTx) Get(typ ResourceType, conds map[string]interface{}) (interface{}, error) {
	return tx.masterTx.Get(typ, conds)
}

func (tx ReplicateTx) Fill(conds map[string]interface{}, out interface{}) error {
	return tx.masterTx.Fill(conds, out)
}

func (tx ReplicateTx) Exists(typ ResourceType, conds map[string]interface{}) (bool, error) {
	return tx.masterTx.Exists(typ, conds)
}

func (tx ReplicateTx) Count(typ ResourceType, conds map[string]interface{}) (int64, error) {
	return tx.masterTx.Count(typ, conds)
}

func (tx ReplicateTx) CountEx(typ ResourceType, sql string, params ...interface{}) (int64, error) {
	return tx.masterTx.CountEx(typ, sql, params...)
}

func (tx ReplicateTx) Update(typ ResourceType, nv map[string]interface{}, conds map[string]interface{}) (n int64, err error) {
	n, err = tx.masterTx.Update(typ, nv, conds)
	if n == 0 || err != nil {
		return
	}

	n, err = tx.slaveTx.Update(typ, nv, conds)
	return
}

func (tx ReplicateTx) Delete(typ ResourceType, conds map[string]interface{}) (n int64, err error) {
	n, err = tx.masterTx.Delete(typ, conds)
	if n == 0 || err != nil {
		return
	}

	return tx.slaveTx.Delete(typ, conds)
}

func (tx ReplicateTx) GetEx(typ ResourceType, sql string, params ...interface{}) (interface{}, error) {
	return tx.masterTx.GetEx(typ, sql, params...)
}

func (tx ReplicateTx) FillEx(out interface{}, sql string, params ...interface{}) error {
	return tx.masterTx.FillEx(out, sql, params...)
}

func (tx ReplicateTx) Exec(sql string, params ...interface{}) (n int64, err error) {
	n, err = tx.masterTx.Exec(sql, params...)
	if n == 0 || err != nil {
		return
	}

	return tx.slaveTx.Exec(sql, params...)
}

func (tx ReplicateTx) Commit() error {
	masterErr := tx.masterTx.Commit()
	slaveErr := tx.slaveTx.Commit()
	if masterErr != nil {
		return masterErr
	} else if slaveErr != nil {
		return slaveErr
	} else {
		return nil
	}
}

func (tx ReplicateTx) Rollback() error {
	masterErr := tx.masterTx.Rollback()
	slaveErr := tx.slaveTx.Rollback()
	if masterErr != nil {
		return masterErr
	} else if slaveErr != nil {
		return slaveErr
	} else {
		return nil
	}
}
