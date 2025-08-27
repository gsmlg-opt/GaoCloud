# key Value高可用存储
## 概要
通过部署多个存储实例，数据更改同步更改所有数据库的方式，实现kv数据库的高可用

## 动机和目标
所有状态都需要考虑高可用，单机故障已经是大概率事件，同时定期的数据备份会带来大量
的运维开销，所以高可用的数据解决方案已经是存储的基本需求。

针对某些对数据的写入和读取频繁不高的应用，同时不考虑跨数据中心的情况，考虑使用最简单的，
同时写入多个副本的设计，当某个存储节点宕机，可以从其他节点通过同步文件的方式同步数据。

## 架构
```text
                      +-------------------+
+------------+   +--->|kvdb server(master)|
|app         |   |    +-------------------+
|   +-------+|   |    +-------------------+
|   |client |<---|--->|kvdb server(slave) |
|   +-------+|   |    +-------------------+
+------------+   |    +-------------------+
                 +--->|kvdb server(slave) |
                      +-------------------+
```
所有kv数据库服务器是完全等同的，没有差别
应用使用提供的client接口访问所有的kv数据库
为了处理错误方便，client指定其中某个kv服务器为master，其他的服务器都是slave
- 读取操作
  client只从master节点读取数据，如果失败直接报错
- 更新操作
  任何更改操作，首先写入master，如果更新失败，client接口报错
  更新完master之后，逐一更新slave，更新失败会记录日志，但是不会报错

### kv数据库
每个kv服务默认使用boltdb作为存储引擎，存储接口
```go
type DB interface {
    Chechsum() (string, error)
    //Close and Destroy are mutually exclusive
    //release the conn
    Close() error
    //clean all the data and release the conn
    Destroy() error

    //like path, create child table, will create all parent table too
    CreateOrGetTable(TableName) (Table, error)
    //delete parent table will delete all child table
    DeleteTable(TableName) error
}

type Table interface {
    Begin() (Transaction, error)
}

type Transaction interface {
    Commit() error
    Rollback() error

    Add(string, []byte) error
    //delete non-exist key returns nil
    Delete(string) error
    Update(string, []byte) error
    //get non-exist key return err
    Get(string) ([]byte, error)
    List() (map[string][]byte, error)
}
```
-  表的名字类似文件路径，删除一级table，如同删除父目录一样会自动删除所有子表，这样的设计
便于处理资源父子关系，当删除父资源，子资源自动删除
-  对于transaction，如果commit成功之后再次调用rollback将不起任何作用，这样设计方便实用go的defer语法。
-  所有的数据保存在一个文件中, 便于数据导入和导出。

### kv服务器
kv服务器使用grpc协议，client屏蔽服务器的一切协议交互，同时client实现了db接口，使得应用访问
远端数据库如同访问本地服务一样

## 数据一致性保证
client在启动的时候，会去获取所有节点数据的checksum值，并进行对比，如果checksum值不一致，client会报错。
从而保证当系统发送变化，重新启动的时候，各节点的数据总是一致的。

## 未来工作
- 当某个节点宕机，数据导入需要停止整个应用，然后采用拷贝文件的方式来同步数据
- master节点的宕机会让应用报错，处理方式有
  - 恢复master节点，把数据从slave节点拷贝过来
  - 重新启动应用，把某个slave节点提升成master节点
  这两种方案，都需要手动干预
