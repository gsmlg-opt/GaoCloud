package db

import (
	"fmt"
	"reflect"

	"gorest/resource"
)

type ResourceStore interface {
	//clear all the data
	Clean()
	//close the conn to db
	Close()

	Begin() (Transaction, error)
}

type Transaction interface {
	Insert(r resource.Resource) (resource.Resource, error)
	//return an slice of Resource which is a pointer to struct
	Get(typ ResourceType, cond map[string]interface{}) (interface{}, error)
	//this is used for many to many relationship
	//which means there is a seperate table which owner and owned resource in it
	//return an slice of Resource which is a pointer to struct
	GetOwned(owner ResourceType, ownerID string, owned ResourceType) (interface{}, error)
	Exists(typ ResourceType, cond map[string]interface{}) (bool, error)
	Count(typ ResourceType, cond map[string]interface{}) (int64, error)
	//out should be an slice of Resource which is a pointer to struct
	Fill(cond map[string]interface{}, out interface{}) error
	Delete(typ ResourceType, cond map[string]interface{}) (int64, error)
	Update(typ ResourceType, nv map[string]interface{}, cond map[string]interface{}) (int64, error)
	//Samilar with GetOwned
	//out should be an slice of Resource which is a pointer to struct
	FillOwned(owner ResourceType, ownerID string, out interface{}) error

	GetEx(typ ResourceType, sql string, params ...interface{}) (interface{}, error)
	CountEx(typ ResourceType, sql string, params ...interface{}) (int64, error)
	FillEx(out interface{}, sql string, params ...interface{}) error
	Exec(sql string, params ...interface{}) (int64, error)

	Commit() error
	Rollback() error
}

func WithTx(store ResourceStore, f func(Transaction) error) error {
	tx, err := store.Begin()
	if err == nil {
		err = f(tx)
		if err == nil {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}
	return err
}

//out should be a slice of struct pointer
func GetResourceWithID(store ResourceStore, id string, out interface{}) (interface{}, error) {
	err := WithTx(store, func(tx Transaction) error {
		return tx.Fill(map[string]interface{}{IDField: id}, out)
	})
	if err != nil {
		return nil, err
	}

	sliceVal := reflect.ValueOf(out).Elem()
	if sliceVal.Len() == 1 {
		return sliceVal.Index(0).Interface(), nil
	} else {
		return nil, fmt.Errorf("not found")
	}
}
