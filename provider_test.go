package txnp

import (
	"context"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
)

var (
	TxnProvider1 Provider
	TxnProvider2 Provider
)

func newGormDBAndTxnProvider(dsn string) (*gorm.DB, Provider, error) {
	conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	// 注册插件
	err = conn.Use(&GormTxnPlugin{})
	if err != nil {
		return nil, nil, err
	}
	// 注入db，返回provider
	return conn, NewTransactionProvider(conn), nil
}

type TestPo struct {
	ID       int64  `gorm:"column:id;type:int(11) unsigned;primary_key;AUTO_INCREMENT"`
	TestName string `gorm:"column:test_name"`
}

func (TestPo) TableName() string {
	return "test_po"
}

// TestTransactionImpl_TransactionExample 测试事务，一个使用的栗子
func TestTransactionImpl_TransactionExample(t *testing.T) {
	// mysql的测试, 本地的链接
	localDsn := "root:123456@tcp(127.0.0.1:3306)/local_test?charset=utf8mb4"
	var (
		conn1 *gorm.DB
		conn2 *gorm.DB
		err   error
	)
	conn1, TxnProvider1, err = newGormDBAndTxnProvider(localDsn)
	require.Nil(t, err)
	conn2, TxnProvider2, err = newGormDBAndTxnProvider(localDsn)
	require.Nil(t, err)

	if isExisted := conn1.Migrator().HasTable("test_po"); isExisted {
		err = conn1.Migrator().DropTable(&TestPo{})
		require.Nil(t, err)
	}
	err = conn1.Migrator().CreateTable(&TestPo{})

	ctx := context.Background()

	err = TxnProvider2.Transaction(ctx, func(ctx context.Context) error {
		e := conn2.WithContext(ctx).Create(&TestPo{TestName: "test1"}).Error // 落库成功
		if e != nil {
			return errors.Wrap(e, "insert a error")
		}
		e = conn2.WithContext(ctx).Create(&TestPo{TestName: "test11"}).Error // 落库成功
		if e != nil {
			return errors.Wrap(e, "insert b error")
		}
		return nil
	})
	require.Nil(t, err)

	// 嵌套成功
	err = TxnProvider1.Transaction(ctx, func(ctx context.Context) error {
		e := conn1.WithContext(ctx).Create(&TestPo{TestName: "test2"}).Error //落库成功
		if e != nil {
			return errors.Wrap(e, "insert a error")
		}
		return TxnProvider1.Transaction(ctx, func(ctx context.Context) error {
			e = conn1.WithContext(ctx).Create(&TestPo{TestName: "test22"}).Error //落库成功
			if e != nil {
				return errors.Wrap(e, "insert b error")
			}
			return nil
		})
	})
	require.Nil(t, err)

	// 嵌套，上层失败
	err = TxnProvider1.Transaction(ctx, func(ctx context.Context) error {
		e := conn1.WithContext(ctx).Create(&TestPo{TestName: "test3"}).Error //落库失败
		if e != nil {
			return errors.Wrap(e, "insert a error")
		}
		_ = TxnProvider1.Transaction(ctx, func(ctx context.Context) error {
			e = conn1.WithContext(ctx).Create(&TestPo{TestName: "test33"}).Error //落库失败
			if e != nil {
				return errors.Wrap(e, "insert b error")
			}
			return nil
		})
		return errors.New("first rollback")
	})
	require.NotNil(t, err)

	// 不同db,不同provider
	err = TxnProvider1.Transaction(ctx, func(ctx context.Context) error {
		e := conn1.WithContext(ctx).Create(&TestPo{TestName: "test4"}).Error // 落库失败
		if e != nil {
			return errors.Wrap(e, "insert a error")
		}
		e = TxnProvider2.Transaction(ctx, func(ctx context.Context) error {
			e = conn2.WithContext(ctx).Create(&TestPo{TestName: "test44"}).Error // 成功落库
			if e != nil {
				return errors.Wrap(e, "insert b error")
			}
			return nil
		})
		return errors.New("provider1 rollback")
	})
	require.NotNil(t, err)
	// 不用Transaction
	err = conn1.WithContext(ctx).Create(&TestPo{TestName: "test-no transaction"}).Error // 落库成功
	require.Nil(t, err)

}
