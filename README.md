# gorm 事务插件支持

# 背景

gorm 提供了事务的方法
gorm.DB.Transaction(fc func(tx *DB) error, opts ...*sql.TxOptions) (err error)
这个方法不可重用，嵌套起来也比较麻烦，提供一个插件将事务txn注入到context中，方便再同一个上下文中传递
可以更加的关注业务上面的逻辑，而不是关注事务的传递

# 使用说明
每创建一个gorm.DB的时候,需要完成两件，
1. 将插件安装到gorm.DB中
2. 使用gorm.DB去创建一个provider

```go
package main


var TxnProvider1 Provider

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
```
在需要使用的时候，直接使用全局变量TxnProvider1.Transaction()即可

# 例子
[TestTransactionImpl_TransactionExample](provider_test.go)

# 注意事项
1. 事务的传递是通过context传递的，所以在使用的时候，需要注意context的传递
2. 事务的gorm.DB在context的key是初始化注入的grom.DB的地址值,所以不同的gorm.DB不会是同一个事务
3. Transaction会创建一个新的context，Transaction里面下面调用的conetxt可能已经改变了
