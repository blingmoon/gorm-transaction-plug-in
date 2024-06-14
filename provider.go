package txnp

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"runtime"
)

// 出于对事务处理，将多个表的事务同时处理

// Provider 主要是为了service可以方便的调用事务而做出来的一个结构体
type Provider interface {
	//
	// Transaction
	//  @Description: 开启一个非自动提交的事务, 可以嵌套调用，每次都会从ctx中找是否有合适的事务，如果没有
	//  			  合适的则会新建一个事务，并且注入到ctx，开启一个事务，任意一个fc抛出错误都会进行回滚
	//  @param ctx 上下文，会尝试从ctx中获取事务
	//  @param fc 需要执行的动作，如果返回error，就会直接回滚
	//  @return error
	//
	Transaction(ctx context.Context, fc func(ctx context.Context) error) error
}
type transactionImpl struct {
	db *gorm.DB
}

func NewTransactionProvider(db *gorm.DB) Provider {
	return &transactionImpl{
		db: db,
	}
}

func (t *transactionImpl) Transaction(ctx context.Context, fc func(ctx context.Context) error) (err error) {
	txn, ok := t.getTxnFromContext(ctx)
	if ok {
		// 有事务了，继续执行就好了
		return fc(ctx)
	}
	if t.db == nil {
		return fmt.Errorf("mysql client is nil")
	}
	txn = t.db.Begin()
	defer func() {
		if e := recover(); e != nil {
			buf := make([]byte, 4096)
			buf = buf[:runtime.Stack(buf, false)]
			err = errors.Errorf("db transaction panic: %v, stack: \n%s", e, buf)
		}
		if err != nil {
			_ = txn.Rollback().Error
		}
	}()

	newctx := context.WithValue(ctx, getDbTxnID(t.db), txn)
	err = fc(newctx)
	if err == nil {
		err = txn.Commit().Error
	}
	return
}

func (t *transactionImpl) getTxnFromContext(ctx context.Context) (*gorm.DB, bool) {
	txAny := ctx.Value(getDbTxnID(t.db))
	tx, ok := txAny.(*gorm.DB)
	if !ok {
		return nil, false
	}
	return tx, true
}
