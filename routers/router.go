package routers

import (
	"dispatch/controllers"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/", &controllers.MainController{})
	dubbo := beego.NewNamespace("/dubbo",
		beego.NSRouter("/getflow", &controllers.DubboController{}, "get:GetFlowFromHost"),
		beego.NSRouter("/modflow", &controllers.DubboController{}, "post:ModFlowFromHost"),
	)
	beego.AddNamespace(dubbo)
}
