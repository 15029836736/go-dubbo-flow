package controllers

import (
	"dispatch/utils"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/astaxie/beego"
)

type DubboPostBody struct {
	Host    string `json:"host"`
	Env     string `json:"env"`
	Weight  int    `json:"weight"`
	Disable bool   `json:"disable"`
}

type DubboController struct {
	beego.Controller
}

//get 查看host对应的dubbo流量情况
func (c *DubboController) GetFlowFromHost() {
	host := c.Input().Get("host")
	env := c.Input().Get("env")
	if host == "" {
		c.Ctx.WriteString("{\"ok\": false,\"errorCode\": 1, \"errorMsg\":\"没有输入host参数\"}")
		return
	}
	if env == "" {
		c.Ctx.WriteString("{\"ok\": false,\"errorCode\": 1, \"errorMsg\":\"没有输入env参数\"}")
		return
	}
	if _, ok := utils.ZkConn[env]; !ok {
		c.Ctx.WriteString("{\"ok\": fasle,\"errorCode\": 1, \"errorMsg\": \"当前运行的环境不存在\"}")
		return
	}
	srcdata := utils.WeightGet(utils.ZkConn[env], utils.ZkMap[env], strings.Split(host, ","))
	//fmt.Println(srcdata)
	resdata, err := json.Marshal(srcdata)
	if err != nil {
		c.Ctx.WriteString(fmt.Sprintf("{\"ok\": false,\"errorCode\": 1, \"errorMsg\":\"%v\"}", err))
		return
	}
	c.Ctx.WriteString(fmt.Sprintf("{\"ok\": true,\"errorCode\": 0, \"errorMsg\":null, \"data\":%s}", string(resdata)))
}

//post 修改host对应的dubbo流量情况
func (c *DubboController) ModFlowFromHost() {
	var postbody DubboPostBody
	err := json.Unmarshal(c.Ctx.Input.RequestBody, &postbody)
	if err != nil {
		c.Ctx.WriteString(fmt.Sprintf("{\"ok\": false,\"errorCode\": 1, \"errorMsg\":\"%v\"}", err))
		return
	}
	if postbody.Host == "" {
		c.Ctx.WriteString("{\"ok\": false,\"errorCode\": 1, \"errorMsg\":\"没有输入host参数\"}")
		return
	}
	if postbody.Env == "" {
		c.Ctx.WriteString("{\"ok\": false,\"errorCode\": 1, \"errorMsg\":\"没有输入env参数\"}")
		return
	}
	if postbody.Weight > 1000 {
		postbody.Weight = 1000
	}
	if postbody.Weight <= 0 {
		postbody.Weight = 0
		postbody.Disable = true
	}
	//fmt.Println(postbody)
	if _, ok := utils.ZkConn[postbody.Env]; !ok {
		c.Ctx.WriteString("{\"ok\": false,\"errorCode\": 1, \"errorMsg\": \"当前运行的环境不存在\"}")
		return
	}
	err = utils.WeightMod(postbody.Host, utils.ZkConn[postbody.Env], utils.ZkMap[postbody.Env], postbody.Weight, postbody.Disable)
	if err != nil {
		c.Ctx.WriteString(fmt.Sprintf("{\"ok\": false,\"errorCode\": 1, \"errorMsg\": \"创建zk节点失败,原因:%v\"}", err))
		return
	}
	c.Ctx.WriteString("{\"ok\": true,\"errorCode\": 0, \"errorMsg\":null}")
}
