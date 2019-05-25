# go-dubbo-flow
公司使用的dubbo-admin版本太老，没有对应的api；新版本admin又存在不少bug没有正式启用。老大希望5月时候将http权重控制和dubbo权重控制放到服务树上，故用go语言 beego框架写了操作dubbo流量权重的程序，简单粗暴，直接操作zk。后期会加上访问控制和单服务的权重修改。

一、使用方法


1.获取某个ip:port(实例)下所有服务的权重


host参数是ip:port，如果有多个可以用,隔开


参数env表示选择哪一个环境的dubbo，一个公司必定存在多个环境，可以多环境使用


api:/dubbo/getflow


param:host=${ip:port}&env=${env}


2.修改某个ip:port(实例)下所有服务的权重


传json header，body体如下，weight权重是int整型，0代表禁用


api:/dubbo/modflow


body:{host:${ip:port}, env:${env}, weight:${int}}
