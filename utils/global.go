package utils

import (
	"fmt"
	"sync"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/samuel/go-zookeeper/zk"
)

//反解dubbo结构，ip:[func1,func2,...]
type FuncMap map[string][]string

//返回查询结果的结构体
type Result struct {
	Weight  int  `json:"weight"`
	Disable bool `json:"disable"`
}

//查询dubbo结构体
type ResFind map[string]map[string]Result

//zk连接情况
type dubbozk *zk.Conn

var ZkMap map[string]FuncMap
var ZkConn map[string]dubbozk

//全局锁
var Mutex *sync.Mutex

//日志
var applog *logs.BeeLogger

func Init() {
	Mutex = new(sync.Mutex)
	ZkMap = map[string]FuncMap{}
	ZkConn = map[string]dubbozk{}
	applog = logs.NewLogger()
	applog.SetLogger(logs.AdapterFile, fmt.Sprintf(`{"filename":"%s"}`, beego.AppConfig.String("logs")))
	applog.EnableFuncCallDepth(true)
}
