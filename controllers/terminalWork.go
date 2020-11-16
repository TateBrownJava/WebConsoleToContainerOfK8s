package controllers

import "github.com/astaxie/beego"

type TerminalController struct {
	beego.Controller
}

func (self *TerminalController) Get() {
	//环境信息
	self.Data["context"] = self.GetString("context")
	self.Data["namespace"] = self.GetString("namespace")
	self.Data["pod"] = self.GetString("pod")
	self.Data["container"] = self.GetString("container")
	self.TplName = "work.html"
}