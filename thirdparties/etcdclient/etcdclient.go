package main

/*
#include <stdlib.h>

enum EtcdErrCode
{
    // grpc errCode, 具体的含义见:
    // https://godoc.org/go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes#ErrGRPCNoSpace
    // https://godoc.org/google.golang.org/grpc/codes#Code
    OK = 0,
    Canceled = 1,
    Unknown = 2,
    InvalidArgument = 3,
    DeadlineExceeded = 4,
    NotFound = 5,
    AlreadyExists = 6,
    PermissionDenied = 7,
    ResourceExhausted = 8,
    FailedPrecondition = 9,
    Aborted = 10,
    OutOfRange = 11,
    Unimplemented = 12,
    Internal = 13,
    Unavailable = 14,
    DataLoss = 15,
    Unauthenticated = 16,

    // 自定义错误码
    TxnUnkownOp = 17,
    ObjectNotExist = 18,
    ErrObjectType = 19,
    KeyNotExist = 20,
    CampaignInternalErr = 21,
    CampaignLeaderSuccess = 22,
    ObserverLeaderInternal = 23,
    ObserverLeaderChange = 24,
    LeaderResignErr = 25,
    LeaderResiginSuccess = 26
};

enum OpType {
  OpPut = 1,
  OpDelete = 2
};

struct EtcdConf {
    char *Endpoints;
    int len;
    int DialTimeout;
};

struct Operation {
    enum OpType opType;
    char *key;
    char *value;
    int keyLen;
    int valueLen;
};
*/
import "C"
import (
    "context"
    "fmt"
    "errors"
    "strings"
    "time"
    "go.etcd.io/etcd/clientv3"
    "go.etcd.io/etcd/clientv3/concurrency"
    "go.etcd.io/etcd/etcdserver/api/v3rpc/rpctypes"
    mvccpb "go.etcd.io/etcd/mvcc/mvccpb"

    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

const (
    EtcdNewClient = "NewCient"
    EtcdPut = "Put"
    EtcdGet = "Get"
    EtcdList = "List"
    EtcdDelete = "Delete"
    EtcdTxn2 = "Txn2"
    EtcdTxn3 = "Txn3"
    EtcdCmpAndSwp = "CmpAndSwp"
)

var globalClient *clientv3.Client

func GetEndpoint(endpoints string) []string {
    sub := strings.Split(endpoints, ",")
    res := make([]string, 0, len(sub))
    for _, elem := range sub {
        res = append(res, elem)
    }
    return res
}

func GenOpList(cops []C.struct_Operation) ([]clientv3.Op, error) {
    res := make([]clientv3.Op, 0, len(cops))
    for _, op := range cops {
        switch op.opType {
            case C.OpPut:
                goKey := C.GoStringN(op.key, op.keyLen)
                goValue := C.GoStringN(op.value, op.valueLen)
                res = append(res, clientv3.OpPut(goKey, goValue))
            case C.OpDelete:
                goKey := C.GoStringN(op.key, op.keyLen)
                res = append(res, clientv3.OpDelete(goKey))
            default:
                fmt.Printf("opType:%v do not exist", op.opType)
                return res, errors.New("opType do not exist")
        }
    }
    return res, nil
}

func GetErrCode(op string, err error) C.enum_EtcdErrCode {
    errCode := codes.Unknown
    if entity, ok := err.(rpctypes.EtcdError); ok {
        errCode = entity.Code()
    } else if ev, ok := status.FromError(err); ok {
        errCode = ev.Code()
    } else if err == context.Canceled {
        errCode = codes.Canceled
    } else if err == context.DeadlineExceeded {
        errCode = codes.DeadlineExceeded
    }

    if codes.OK != errCode {
        fmt.Printf("etcd do %v get err:%v, errCode:%v\n", op, err, errCode)
    }

    switch errCode {
    case codes.OK:
        return C.OK
    case codes.Canceled:
        return C.Canceled
    case codes.Unknown:
        return C.Unknown
    case codes.InvalidArgument:
        return C.InvalidArgument
    case codes.DeadlineExceeded:
        return C.DeadlineExceeded
    case codes.NotFound:
        return C.NotFound
    case codes.AlreadyExists:
        return C.AlreadyExists
    case codes.PermissionDenied:
        return C.PermissionDenied
    case codes.ResourceExhausted:
        return C.ResourceExhausted
    case codes.FailedPrecondition:
        return C.FailedPrecondition
    case codes.Aborted:
        return C.Aborted
    case codes.OutOfRange:
        return C.OutOfRange
    case codes.Unimplemented:
        return C.Unimplemented
    case codes.Internal:
        return C.Internal
    case codes.Unavailable:
        return C.Unavailable
    case codes.DataLoss:
        return C.DataLoss
    case codes.Unauthenticated:
        return C.Unauthenticated
    }

    return C.Unknown
}

// TODO(lixiaocui): 日志打印看是否需要glog
//export NewEtcdClientV3
func NewEtcdClientV3(conf C.struct_EtcdConf) C.enum_EtcdErrCode {
    var err error
    globalClient, err = clientv3.New(clientv3.Config{
        Endpoints:   GetEndpoint(C.GoStringN(conf.Endpoints, conf.len)),
        DialTimeout: time.Duration(int(conf.DialTimeout)) * time.Millisecond,
    })
    return GetErrCode(EtcdNewClient, err)
}

//export EtcdCloseClient
func EtcdCloseClient() {
    if (globalClient != nil) {
        globalClient.Close()
    }
}

//export EtcdClientPut
func EtcdClientPut(timeout C.int, key, value *C.char,
    keyLen, valueLen C.int) C.enum_EtcdErrCode {
    goKey, goValue := C.GoStringN(key, keyLen), C.GoStringN(value, valueLen)
    ctx, cancel := context.WithTimeout(context.Background(),
        time.Duration(int(timeout))*time.Millisecond)
    defer cancel()

    _, err := globalClient.Put(ctx, goKey, goValue)
    return GetErrCode(EtcdPut, err)
}

//export EtcdClientGet
func EtcdClientGet(timeout C.int, key *C.char,
    keyLen C.int) (C.enum_EtcdErrCode, *C.char, int) {
    goKey := C.GoStringN(key, keyLen)
    ctx, cancel := context.WithTimeout(context.Background(),
        time.Duration(int(timeout))*time.Millisecond)
    defer cancel()

    resp, err := globalClient.Get(ctx, goKey)
    errCode := GetErrCode(EtcdGet, err)
    if errCode != C.OK {
        return errCode, nil, 0
    }

    if resp.Count <= 0 {
        return C.KeyNotExist, nil, 0
    }

    return errCode,
           C.CString(string(resp.Kvs[0].Value)),
           len(resp.Kvs[0].Value)
}

// TODO(lixiaocui): list可能需要有长度限制
//export EtcdClientList
func EtcdClientList(timeout C.int, startKey, endKey *C.char,
    startLen, endLen C.int) (C.enum_EtcdErrCode, uint64, int64) {
    goStartKey := C.GoStringN(startKey, startLen)
    goEndKey := C.GoStringN(endKey, endLen)
    ctx, cancel := context.WithTimeout(context.Background(),
        time.Duration(int(timeout)) * time.Millisecond)
    defer cancel()

    var resp *clientv3.GetResponse
    var err error
    if goEndKey == "" {
        // return keys >= start
        resp, err = globalClient.Get(
            ctx, goStartKey, clientv3.WithFromKey());
    } else {
        // return keys in range [start, end)
        resp, err = globalClient.Get(
            ctx, goStartKey, clientv3.WithRange(goEndKey))
    }

    errCode := GetErrCode(EtcdList, err)
    if errCode != C.OK {
        return errCode, 0, 0
    }
    return  errCode, AddManagedObject(resp.Kvs), resp.Count
}

//export EtcdClientDelete
func EtcdClientDelete(
    timeout C.int, key *C.char, keyLen C.int) C.enum_EtcdErrCode {
    goKey := C.GoStringN(key, keyLen)
    ctx, cancel := context.WithTimeout(context.Background(),
        time.Duration(int(timeout))*time.Millisecond)
    defer cancel()

    _, err := globalClient.Delete(ctx, goKey)
    return GetErrCode(EtcdDelete, err)
}

//export EtcdClientTxn2
func EtcdClientTxn2(
    timeout C.int, op1, op2 C.struct_Operation) C.enum_EtcdErrCode {
    ops := []C.struct_Operation{op1, op2}
    etcdOps, err := GenOpList(ops)
    if err != nil {
        fmt.Printf("unknown op types, err: %v\n", err)
        return C.TxnUnkownOp
    }

    ctx, cancel := context.WithTimeout(context.Background(),
        time.Duration(int(timeout))*time.Millisecond)
    defer cancel()

    _, err = globalClient.Txn(ctx).Then(etcdOps...).Commit()
    return GetErrCode(EtcdTxn2, err)
}

//export EtcdClientTxn3
func EtcdClientTxn3(
    timeout C.int, op1, op2, op3 C.struct_Operation) C.enum_EtcdErrCode {
    ops := []C.struct_Operation{op1, op2, op3}
    etcdOps, err := GenOpList(ops)
    if (err != nil) {
        fmt.Printf("unknown op types, err: %v\n", err)
        return C.TxnUnkownOp
    }

    ctx, cancel := context.WithTimeout(context.Background(),
        time.Duration(int(timeout))*time.Millisecond)
    defer cancel()

    _, err = globalClient.Txn(ctx).Then(etcdOps...).Commit()
    return GetErrCode(EtcdTxn3, err)
}

//export EtcdClientCompareAndSwap
func EtcdClientCompareAndSwap(timeout C.int, key, prev, target *C.char,
    keyLen, preLen, targetLen C.int) C.enum_EtcdErrCode {
    goKey := C.GoStringN(key, keyLen)
    goPrev := C.GoStringN(prev, preLen)
    goTarget := C.GoStringN(target, targetLen)

    ctx, cancel := context.WithTimeout(context.Background(),
    time.Duration(int(timeout))*time.Millisecond)
    defer cancel()

    var err error
    _, err = globalClient.Txn(ctx).
        If(clientv3.Compare(clientv3.CreateRevision(goKey), "=", 0)).
        Then(clientv3.OpPut(goKey, goTarget)).
        Commit()

    _, err = globalClient.Txn(ctx).
        If(clientv3.Compare(clientv3.Value(goKey), "=", goPrev)).
        Then(clientv3.OpPut(goKey, goTarget)).
        Commit()
    return GetErrCode(EtcdCmpAndSwp, err)
}

//export EtcdElectionCampaign
func EtcdElectionCampaign(pfx *C.char, pfxLen C.int,
    leaderName *C.char, nameLen C.int, sessionInterSec uint32,
    electionTimeoutMs uint32) (C.enum_EtcdErrCode, uint64) {
    // TODO(lixiaocui):  MDS的切换时间是否能够控制在 ms级别，比如500ms以内
    goPfx := C.GoStringN(pfx, pfxLen)
    goLeaderName := C.GoStringN(leaderName, nameLen)

    // 创建带ttl的session
    var sessionOpts concurrency.SessionOption =
            concurrency.WithTTL(int(sessionInterSec))
    session, err := concurrency.NewSession(globalClient, sessionOpts)
    if err != nil {
        fmt.Printf("%v new session err: %v\n", goLeaderName, err)
        return C.CampaignInternalErr, 0
    }

    // 创建election和超时context
    var election *concurrency.Election = concurrency.NewElection(session, goPfx)
    var ctx context.Context
    var cancel context.CancelFunc
    if electionTimeoutMs > 0 {
        ctx, cancel = context.WithTimeout(context.Background(),
            time.Duration(int(electionTimeoutMs)) * time.Millisecond)
        defer cancel()
    } else {
        ctx = context.Background()
    }

    // 获取当前leader并打印
    if leader, err := election.Leader(ctx); err == nil {
        fmt.Printf("current leader is: %v\n", string(leader.Kvs[0].Value))
    } else {
        fmt.Printf("get current leader err: %v\n", err)
    }

    // 如果contex是'context.TODO()/context.Background()', 竞选会阻塞到该key被删除,
    // 除非返回non-recoverable error(例如: ErrCompacted).
    // 如果context有超时, 在超时或者取消之前, 竞选会阻塞到当选leader
    fmt.Printf("%v campagin for leader begin\n", goLeaderName)
    if err := election.Campaign(ctx, goLeaderName); err != nil {
        fmt.Printf("%v campaign err: %v\n", goLeaderName, err)
        return C.CampaignInternalErr, 0
    } else {
        fmt.Printf("%v campaign for leader success\n", goLeaderName)
    }
    return C.CampaignLeaderSuccess, AddManagedObject(election)
}

//export EtcdLeaderObserve
func EtcdLeaderObserve(leaderOid uint64, timeout uint64,
    leaderName *C.char, nameLen C.int) C.enum_EtcdErrCode {
    goLeaderName := C.GoStringN(leaderName, nameLen)

    election := GetLeaderElection(leaderOid)
    if election == nil {
        fmt.Printf("can not get leader object: %v\n", leaderOid)
        return C.ObjectNotExist
    }

    fmt.Printf("start observe: %v\n", goLeaderName)
    var ctx context.Context
    ctx = clientv3.WithRequireLeader(context.Background())

    ticker := time.NewTicker(time.Duration(timeout / 5) * time.Millisecond)
    defer ticker.Stop()

    observer := election.Observe(ctx)
    for {
        select {
        case resp, ok := <-observer:
            if !ok {
                fmt.Printf("Observe() channel closed permaturely\n")
                return C.ObserverLeaderInternal
            }
            if string(resp.Kvs[0].Value) == goLeaderName {
                continue
            }
            fmt.Printf("Observe() leaderChange, now is: %v, expect: %v\n",
                resp.Kvs[0].Value, goLeaderName)
            return C.ObserverLeaderChange
        // observe如果设置有timeout的context, 含义是observe timeout时间便退出，其次，
        // 还是get操作的超时时间。
        // 在应用中希望对该key一直进行检测，因此context不带超时。这样会导致etcd挂掉的时候
        // grpc无限重试，所以需要在外部定时去检查etcd网络连接状态
        case <-ticker.C:
            // 定期和mds通信，确认网络是否正常
            t := time.Now()
            ctx, cancel := context.WithTimeout(context.Background(),
                time.Duration(int(timeout))*time.Millisecond)
            defer cancel()

            _, err := globalClient.Get(ctx, "observe-test")
            errCode := GetErrCode(EtcdGet, err)
            if errCode != C.OK {
                fmt.Printf("Observe hung since %v\n", t)
                return C.ObserverLeaderInternal
            }
        }
    }
}

//export EtcdLeaderResign
func EtcdLeaderResign(leaderOid uint64, timeout uint64) C.enum_EtcdErrCode {
    election := GetLeaderElection(leaderOid)
    if election == nil {
        fmt.Printf("can not get leader object: %v\n", leaderOid)
        return C.ObjectNotExist
    }

    ctx, cancel := context.WithTimeout(context.Background(),
        time.Duration(timeout) * time.Millisecond)
    defer cancel()

    var leader *clientv3.GetResponse
    var err error
    if leader, err = election.Leader(ctx); err != nil {
        fmt.Printf("Leader() returned non nil err: %s\n", err)
        return C.LeaderResignErr
    }

    if err := election.Resign(ctx); err != nil {
        fmt.Printf("%v resign leader err: %v\n",
            string(leader.Kvs[0].Value), err)
        return C.LeaderResignErr
    }

    fmt.Printf("%v resign leader success\n", string(leader.Kvs[0].Value))
    return C.LeaderResiginSuccess
}

//export EtcdClientGetSingleObject
func EtcdClientGetSingleObject(
    oid uint64) (C.enum_EtcdErrCode, *C.char, int) {
    if value, exist := GetManagedObject(oid); !exist {
        fmt.Printf("can not get object: %v\n", oid)
        return C.ObjectNotExist, nil, 0
    } else if res, ok := value.([]*mvccpb.KeyValue); ok {
        return C.OK, C.CString(string(res[0].Value)), len(res[0].Value)
    } else {
        fmt.Printf("object type err\n")
        return C.ErrObjectType, nil, 0
    }
}

//export EtcdClientGetMultiObject
func EtcdClientGetMultiObject(
    oid uint64, serial int) (C.enum_EtcdErrCode, *C.char, int) {
    if value, exist := GetManagedObject(oid); !exist {
        return C.ObjectNotExist, nil, 0
    } else if res, ok := value.([]*mvccpb.KeyValue); ok {
        return C.OK, C.CString(string(res[serial].Value)),
            len(res[serial].Value)
    } else {
        return C.ErrObjectType, nil, 0
    }
}

//export EtcdClientRemoveObject
func EtcdClientRemoveObject(oid uint64) {
    RemoveManagedObject(oid)
}

func GetLeaderElection(leaderOid uint64) *concurrency.Election {
    var election *concurrency.Election
    var ok bool
    if value, exist := GetManagedObject(leaderOid); !exist {
        fmt.Printf("can not get leader object: %v\n", leaderOid)
        return nil
    } else if election, ok = value.(*concurrency.Election); !ok {
        fmt.Printf("oid %v does not type of *concurrency.Election\n", leaderOid)
        return nil
    }

    return election
}
func main() {}
