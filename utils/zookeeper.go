package utils

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego"
	"github.com/samuel/go-zookeeper/zk"
)

//连接zk函数
func ConnectZk(objzkdir string) (*zk.Conn, error) {
	hosts := strings.Split(beego.AppConfig.String(objzkdir), ",")

	zkConn, _, err := zk.Connect(hosts, time.Second*3) //连接zk
	if err != nil {
		applog.Error(fmt.Sprintf("连接zk失败，错误原因: %v", err))
		return nil, err
	}
	return zkConn, nil
}

//检测全局zk连接是否存活
func Alive(zkConn *zk.Conn) bool {
	exist, _, err := zkConn.Exists("/")
	if err != nil {
		applog.Error(fmt.Sprintf("zk测活失败，错误原因: %v", err))
		return false
	}
	return exist
}

//反解Zk dubbo目录，返回指定的结构
func ZkResolve(objzkdir string, zkConn *zk.Conn, mutex *sync.Mutex) (FuncMap, error) {
	defer mutex.Unlock()

	funcs, _, err := zkConn.Children(beego.AppConfig.String("zkdir")) //获取得到zkdir目录下所有的子目录，对dubbo而言，就是方法
	if err != nil {
		applog.Error(fmt.Sprintf("获取%s节点子目录失败，错误原因: %v", beego.AppConfig.String("zkdir"), err))
		return nil, err
	}

	funcmap := make(FuncMap)
	for _, cfunc := range funcs {
		hosts, _, err := zkConn.Children(beego.AppConfig.String("zkdir") + "/" + cfunc + "/" + "providers") //获取提供者的子目录
		if err != nil {
			applog.Error(fmt.Sprintf("获取%s节点子目录失败，错误原因: %v", beego.AppConfig.String("zkdir")+"/"+cfunc+"/"+"providers", err))
			return funcmap, err
		}
		for _, info := range hosts {
			deinfo, err := url.QueryUnescape(info) //解码url
			if err != nil {
				applog.Error(fmt.Sprintf("解码url失败,url:%s ,错误原因: %v", info, err))
				return funcmap, err
			}
			host := strings.Split(deinfo, "/")[2]
			version := ""
			params := strings.Split(strings.Split(deinfo, "?")[1], "&")
			for _, param := range params {
				if strings.Index(param, "version=") != -1 { //查看version
					version = strings.Split(param, "=")[1]
					break
				}
			}
			funcmap[host] = append(funcmap[host], cfunc+":"+version) //构造结果集
		}
	}
	return funcmap, nil
}

// 更新
func UpDataTimer(zkConn *zk.Conn, zkmap map[string]FuncMap, env, objzkdir string) {
	for {
		var err error
		if !Alive(zkConn) {
			applog.Error("全局zk连接 异常，尝试重新连接...")
			zkConn, err = ConnectZk(objzkdir)
			if err != nil {
				applog.Error(fmt.Sprintf("全局zk连接重连异常,错误原因:%v", err))
				continue
			}
			applog.Info("全局zk连接重新成功")
		}
		Mutex.Lock() //共享 锁
		zkmap[env], err = ZkResolve(objzkdir, zkConn, Mutex)
		if err != nil {
			applog.Error(fmt.Sprintf("解析dubbo数据失败，错误原因: %v", err))
			continue
		}
		//fmt.Println(funcmap)
		time.Sleep(time.Second * 3)
	}
}

//获取参数
func getparam(url string) Result {
	resdata := Result{Weight: 100, Disable: false}
	if strings.Index(url, "?") == -1 {
		return resdata
	}
	params := strings.Split(strings.Split(url, "?")[1], "&")
	for _, param := range params {
		if strings.Index(param, "weight=") != -1 { //查看weight
			weight, err := strconv.Atoi(strings.Split(param, "=")[1])
			if err != nil {
				continue
			}
			resdata.Weight = weight
		}
		if strings.Index(param, "enable=") != -1 { //查看禁用状态
			if param == "enable=false" {
				resdata.Disable = true
			}
		}
		if strings.Index(param, "disabled=") != -1 { //查看禁用状态
			if param == "disabled=true" {
				resdata.Disable = true
			}
		}
	}
	return resdata
}

//生成动态url
func makeurl(host, function, version string, weight int, disable bool) string {
	srcurl := fmt.Sprintf("override://%s/%s?category=configurators&dynamic=false&enabled=true&version=%s&weight=%d", host, function, version, weight)
	if disable { //如果是禁用
		srcurl = fmt.Sprintf("override://%s/%s?category=configurators&disabled=%v&dynamic=false&enabled=true&version=%s", host, function, disable, version)
	}
	url := url.QueryEscape(srcurl)
	return url
}

//循环删除zk节点
func deletenode(host, dstpath string, zkConn *zk.Conn) {
	infos, _, err := zkConn.Children(dstpath) //获取提供者的子目录
	if err != nil {
		applog.Error(fmt.Sprintf("获取%s节点子目录失败，错误原因: %v", dstpath, err))
		return
	}
	for _, info := range infos {
		deinfo, err := url.QueryUnescape(info) //解码url
		if err != nil {
			applog.Error(fmt.Sprintf("解码url失败,url:%s ,错误原因: %v", info, err))
			continue
		}
		ipport := strings.Split(deinfo, "/")[2]
		if host == ipport { //判断是不是host，如果是,删除这个记录
			_, stat, err := zkConn.Get(dstpath + "/" + info)
			if err != nil {
				applog.Error(fmt.Sprintf("获取zk目录失败,url:%s ,错误原因: %v", dstpath+"/"+info, err))
				continue
			}
			err = zkConn.Delete(dstpath+"/"+info, stat.Version) //删除zk
			if err != nil {
				applog.Error(fmt.Sprintf("删除zk目录失败,url:%s ,错误原因: %v", dstpath+"/"+info, err))
				continue
			}
		}
	}
}

//权重查询,返回该host所有的服务状态
func WeightGet(zkConn *zk.Conn, funcmap FuncMap, hosts []string) ResFind {
	data := ResFind{}
	for _, host := range hosts { //循环机器列表
		if _, ok := funcmap[host]; ok {
			data[host] = map[string]Result{}
			for _, cfunc := range funcmap[host] { //循环每个机器:port对应的方法
				dstpath := beego.AppConfig.String("zkdir") + "/" + strings.Split(cfunc, ":")[0] + "/" + "configurators"
				infos, _, err := zkConn.Children(dstpath) //获取提供者的子目录
				if err != nil {
					applog.Error(fmt.Sprintf("获取%s节点子目录失败，错误原因: %v", dstpath, err))
					continue
				}

				findit := false
				for _, info := range infos {
					deinfo, err := url.QueryUnescape(info) //解码url
					if err != nil {
						applog.Error(fmt.Sprintf("解码url失败,url:%s ,错误原因: %v", info, err))
						continue
					}
					ipport := strings.Split(deinfo, "/")[2]
					if host == ipport { //判断是不是host，如果是,记录下值
						data[host][cfunc] = getparam(deinfo)
						findit = true
						break
					}
				}
				if !findit {
					data[host][cfunc] = Result{Weight: 100, Disable: false} //默认是这样的
				}
			}
		}
	}
	return data
}

//权重修改、禁用、启用
func WeightMod(host string, zkConn *zk.Conn, funcmap FuncMap, weight int, disable bool) error {
	//Mutex.Lock() //锁
	//defer Mutex.Unlock()
	if _, ok := funcmap[host]; ok {
		for _, cfunc := range funcmap[host] {
			dstpath := beego.AppConfig.String("zkdir") + "/" + strings.Split(cfunc, ":")[0] + "/" + "configurators"
			deletenode(host, dstpath, zkConn)

			if disable || weight != 100 {
				createpath := dstpath + "/" + makeurl(host, strings.Split(cfunc, ":")[0], strings.Split(cfunc, ":")[1], weight, disable)
				_, err := zkConn.Create(createpath, []byte(strings.Split(host, ":")[0]), 0, zk.WorldACL(zk.PermAll))
				if err != nil {
					applog.Error(fmt.Sprintf("创建zk失败,url:%s ,错误原因: %v", createpath, err))
					return err
				}
				//applog.Info(fmt.Sprintf("%s Create Successfully", path))
			}
		}
		applog.Info(fmt.Sprintf("Host %s Mod, Wegith:%d, Disable:%v", host, weight, disable))
	} else {
		applog.Error(fmt.Sprintf("Host %s not exist", host))
	}
	return nil
}
