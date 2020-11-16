package routers

import (
	"pass_homework/controllers"
	"github.com/astaxie/beego"
)

//路由声明
func init() {
    beego.Router("/", &controllers.PageController{})
    beego.Router("/work",&controllers.TerminalController{},"get:Get")
    beego.Handler("/work/websocket",&controllers.WebSocketStruct{},true)
}
