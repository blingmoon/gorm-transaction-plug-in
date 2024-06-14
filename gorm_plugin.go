package txnp

import (
	"context"
	"fmt"
	"gorm.io/gorm"
)

// 插件会用到
type dbTxnID string

func getDbTxnID(db *gorm.DB) dbTxnID {
	if db == nil {
		return ""
	}
	return dbTxnID(fmt.Sprintf("%p", db))
}

func getTxnFromContext(ctx context.Context, txnID dbTxnID) (*gorm.DB, bool) {
	txAny := ctx.Value(txnID)
	tx, ok := txAny.(*gorm.DB)
	if !ok {
		return nil, false
	}
	return tx, true
}

const pluginName = "gorm:transaction-plugin"

type GormTxnPlugin struct {
	key dbTxnID
}

func (txn *GormTxnPlugin) Name() string {
	return pluginName
}

func (txn *GormTxnPlugin) Initialize(db *gorm.DB) error {
	txn.key = getDbTxnID(db)
	return txn.registerCallbacks(db)
}

func (txn *GormTxnPlugin) registerCallbacks(db *gorm.DB) error {

	if err := db.Callback().Create().Before("*").Register(pluginName, txn.continueTxn); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("*").Register(pluginName, txn.continueTxn); err != nil {
		return err
	}

	if err := db.Callback().Delete().Before("*").Register(pluginName, txn.continueTxn); err != nil {
		return err
	}
	if err := db.Callback().Raw().Before("*").Register(pluginName, txn.continueTxn); err != nil {
		return err
	}
	if err := db.Callback().Query().Before("*").Register(pluginName, txn.continueTxn); err != nil {
		return err
	}
	if err := db.Callback().Row().Before("*").Register(pluginName, txn.continueTxn); err != nil {
		return err
	}
	return nil
}

func (txn *GormTxnPlugin) continueTxn(db *gorm.DB) {
	if txn.key == "" {
		// 没有初始化，不必理会
		return
	}
	txnDB, ok := getTxnFromContext(db.Statement.Context, txn.key)
	if ok {
		// 有事务走事务
		db.Statement.ConnPool = txnDB.Statement.ConnPool
		return
	}
	//没有事务不操作
	return
}
