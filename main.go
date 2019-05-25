package main

import (
	_ "dispatch/routers"
	"dispatch/utils"
	"fmt"

	"github.com/astaxie/beego"
)

func main() {
	utils.Init()
	var err error
	var envs []string
	if beego.AppConfig.String("runmode") != "online" { //运行环境不是online，则只建立dev zk
		envs = []string{"dev"}
	} else {
		envs = []string{"pre", "online"}
	}
	for i := 0; i < len(envs); i++ { //初始化zk工作
		utils.ZkConn[envs[i]], err = utils.ConnectZk(envs[i] + "zk")
		if err != nil {
			fmt.Println(err)
			return
		}
		go utils.UpDataTimer(utils.ZkConn[envs[i]], utils.ZkMap, envs[i], envs[i]+"zk")
	}
	beego.Run()
}
