package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/docker/docker/pkg/term"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubernetes/pkg/util/interrupt"
	"net/http"
)

//按顺序读入
type WebSocketStruct struct {
	conn sockjs.Session
	sizeChan chan *remotecommand.TerminalSize
	context string
	namespace string
	pod string
	container string
}

//建立k8s连接环境
func buildConfig(context,kubeconfigPath string)(*rest.Config,error)  {
	//通过goclient建立
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
	return config,err
}

//socket处理逻辑
func socketHandler(ws *WebSocketStruct,cmd string) error {
	configPath := beego.AppConfig.String("kubeconfigpath")
	fmt.Print(configPath)
	config, err := buildConfig(ws.context, configPath)
	if err != nil {
		return err
	}

	//初始化http连接设置
	//序列化，api路径，http相关配置
	config.APIPath = "/api"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: scheme.Codecs}
	groupVersion := schema.GroupVersion{
		Group:   "",
		Version: "v1",
	}
	config.GroupVersion = &groupVersion
	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return err
	}
	//访问匿名函数
	fmt.Println(ws.container)
	fmt.Println(ws.namespace)
	fmt.Println(ws.pod)
	fmt.Println(ws.context)

	realHandleFunc := func() error {
		//配置请求
		request := restClient.Post().Resource("pods").Name(ws.pod).Namespace(ws.namespace).SubResource("exec").
			Param("container", ws.container).Param("stdin", "true").
			Param("stdout", "true").Param("stderr", "true").
			Param("command", cmd).Param("tty", "true")
		//设置参数
		request.VersionedParams(
			&v1.PodExecOptions{
				Container: ws.container,
				Command:   []string{},
				Stdin:     true,
				Stdout:    true,
				Stderr:    true,
				TTY:       true,
			},
			scheme.ParameterCodec,
		)
		//执行
		executor, err := remotecommand.NewSPDYExecutor(
			config, http.MethodPost, request.URL(),
		)
		if err != nil {
			return err
		}
		return executor.Stream(remotecommand.StreamOptions{
			Stdin:             ws,
			Stdout:            ws,
			Stderr:            ws,
			Tty:               true,
			TerminalSizeQueue: ws,
		})
	}
	fdInfo, _ := term.GetFdInfo(ws.conn)
	state, err := term.SaveState(fdInfo)
	return interrupt.Chain(nil, func() {
		term.RestoreTerminal(fdInfo,state)
	}).Run(realHandleFunc)
}

func (self WebSocketStruct) Read(p [] byte) (int,error){
	var recv string
	var message map[string]uint16
	recv, err := self.conn.Recv()
	if err != nil {
		return 0,err
	}
	if err := json.Unmarshal([]byte(recv),&message);err != nil{
		return copy(p,recv),nil
	}else{
		self.sizeChan <- &remotecommand.TerminalSize{
			message["cols"],
			message["rows"],
		}
		return 0,nil
	}
}

func (self WebSocketStruct) Write(p []byte)(int,error){
	err := self.conn.Send(string(p))
	return len(p),err
}

//队列
func (self *WebSocketStruct) Next() *remotecommand.TerminalSize{
	size := <-self.sizeChan
	beego.Debug("终端消息过长")
	return size
}

//http接口实现
func (self WebSocketStruct) ServeHTTP(writer http.ResponseWriter,req *http.Request){
	ctx := req.FormValue("context")
	namespace := req.FormValue("namespace")
	pod := req.FormValue("pod")
	container := req.FormValue("container")
	//socket处理逻辑
	webSocketHandler := func(session sockjs.Session){
		terminal := &WebSocketStruct{
			session,make(chan *remotecommand.TerminalSize),ctx,namespace,pod,container,
		}
		if err := socketHandler(terminal,"/bin/bash"); err != nil{
			beego.Error(err)
		}
	}
	//启动
	sockjs.NewHandler("/work/websocket",sockjs.DefaultOptions,webSocketHandler).ServeHTTP(writer,req)
}








