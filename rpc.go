package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

// Service 代表一个类以及它能使用的方法
type Service struct {
	mu      sync.Mutex
	typ     reflect.Type
	rcvr    reflect.Value
	methods map[string]reflect.Method
}

// Server 代表了一个服务器。一个服务器可以注册多个类。
type Server struct {
	mu       sync.Mutex
	services map[string]*Service // 主要是Raft.Append里面的Radft所谓key。
	count    int                 // 不知道这个有没有什么用
}

// MakeNewServer 在server那边创建一个新的server。
func MakeNewServer() *Server {
	res := new(Server)
	res.services = make(map[string]*Service)
	return res
}

// Install 当创建一个新的server以后，通过这个函数来将类装载到里面.
func (server *Server) Install(object interface{}) error {
	server.mu.Lock()
	defer server.mu.Unlock()
	oTyp := reflect.TypeOf(object)
	if oTyp.Kind() == reflect.Ptr {
		oTyp = oTyp.Elem()
	}
	name := oTyp.Name()
	_, ok := server.services[name]
	if ok == true {
		log.Printf("这个类%s已经装载过了\n", name)
		return errors.New("已经装载过类")
	}
	server.services[name] = Register(object)
	return nil
}

// handleRequest 是用来处理rpc请求的。这个是用在ServeHTTP函数里面的。
func (server *Server) handleRequest(request []byte) ([]byte, bool) {
	decBuf := bytes.NewBuffer(request)
	dec := gob.NewDecoder(decBuf)
	reqmessage := &reqMessage{}
	err := dec.Decode(reqmessage)
	if err != nil {
		log.Println("在handleRequest中，decode输入时发生了问题")
		return nil, false
	}
	dot := strings.IndexAny(reqmessage.SrcMethod, ".")
	if dot < 0 {
		log.Println("无法有效处理输入的字符串，是不是输入有问题？")
		log.Println("需要输入的格式是Raft.Append这样的")
		return nil, false
	}
	oName := reqmessage.SrcMethod[:dot]
	oMethod := reqmessage.SrcMethod[dot+1:]
	service, ok := server.services[oName]
	if ok == false {
		log.Println("服务是否未安装？Server找不到需要的object。")
		return nil, false
	}
	args := reqmessage.Data
	res, ok := service.Call(oMethod, args)
	return res, ok
}

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/api" {
		log.Println(r.URL.Path)
		log.Println("需要访问/v1/api来得到接口")
	}
	switch r.Method {
	case "GET":
		io.WriteString(w, "GET 成功\n")
		log.Println("GET 成功")

	case "POST":
		//io.WriteString(w, "POST 成功\n")
		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("无法从POST中获取到信息")
			return
		}
		log.Println("从POST中获得信息了")
		res, ok := server.handleRequest(buf)
		rpyMess := replyMessage{Ok: ok, Data: res}
		replyBuf := bytes.NewBuffer(nil)
		replyEnc := gob.NewEncoder(replyBuf)
		err = replyEnc.Encode(&rpyMess)
		if err != nil {
			log.Println("在最后将ok和数据编码的时候出现了问题。")
			log.Println(err)
			return
		}
		_, err = w.Write(replyBuf.Bytes())
		if err != nil {
			log.Println("无法将结果返回。在最终返回的时候出了问题。")
			return
		}
	}

}

type reqMessage struct {
	SrcMethod string // 类+名字，就像Raft.Append这样子的
	Data      []byte // 还是用[]byte。通用性可能会更好
}

type replyMessage struct {
	Ok   bool
	Data []byte
}

// Client 代表了一个客户端。我们通过这个来和server进行交流。
type Client struct {
	Adress string
}

// SendRequest client通过这个来发送一个请求。
func (end *Client) SendRequest(name string, args interface{}) ([]byte, error) {
	reqMess := reqMessage{}
	reqMess.SrcMethod = name
	argBuf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(argBuf)
	err := enc.Encode(args)
	if err != nil {
		log.Println("无法在客户端处将输入参数二进制化")
		log.Println(err)
		return nil, err
	}
	reqMess.Data = argBuf.Bytes()
	reqBuf := bytes.NewBuffer(nil)
	encReq := gob.NewEncoder(reqBuf)
	err = encReq.Encode(reqMess)
	if err != nil {
		log.Println("无法在客户端处将reqMess二进制化")
		log.Println(err)
		return nil, err
	}
	resp, err := http.Post(end.Adress, "application/x-www-form-urlencoded", reqBuf)
	if err != nil {
		log.Println("客户端连接失败")
		log.Println(err)
		return nil, err
	}
	defer resp.Body.Close()
	resBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("从连接中读取数据时出了问题")
		log.Println(err)
		return nil, err
	}
	return resBuf, nil
}

// Register 是用来将
func Register(rcvr interface{}) *Service {
	res := &Service{}
	res.methods = make(map[string]reflect.Method)
	res.rcvr = reflect.ValueOf(rcvr)
	res.typ = reflect.TypeOf(rcvr)
	num := res.typ.NumMethod()
	for i := 0; i < num; i++ {
		method := res.typ.Method(i)
		mname := method.Name
		res.methods[mname] = method
	}
	return res
}

// Call 是用来执行一个调用的
// 这里还是使用class.method来调用
// Service 基本上只需要method就行
func (srv *Service) Call(name string, args []byte) ([]byte, bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()
	method, ok := srv.methods[name]
	if ok == false {
		log.Println("无法找到对应的方法")
		return nil, false
	}
	inbuf := bytes.NewBuffer(args)
	argsType := method.Type.In(1)
	//replyType := method.Type.Out(0)
	//replyType = replyType.Elem() // 因为默认函数第二个参数要是指针
	argPtr := reflect.New(argsType)
	//replyPtr := reflect.New(replyType)
	resbuf := bytes.NewBuffer(nil)
	dec := gob.NewDecoder(inbuf)
	enc := gob.NewEncoder(resbuf)
	err := dec.Decode(argPtr.Interface())
	if err != nil {
		log.Println("在从args的[]byte中解码出现了问题")
		return nil, false
	}
	function := method.Func
	res := function.Call([]reflect.Value{srv.rcvr, argPtr.Elem()})
	err = enc.Encode(res[0].Interface())
	if err != nil {
		log.Println("xxx")
		log.Println(err)
		return nil, false
	}
	reply := resbuf.Bytes()
	return reply, true

}

// ResultIsOk 返回从Server中返回的结果是不是OK的。
func ResultIsOk(data []byte) bool {
	var res replyMessage
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&res)
	if err != nil {
		log.Fatalln(err)
	}
	return res.Ok
}

// ConvertToNormalType 这个是将最后的返回值转换为需要的结果的。res 应当是指针。
func ConvertToNormalType(data []byte, res interface{}) {
	resTyp := reflect.TypeOf(res)
	if resTyp.Kind() != reflect.Ptr {
		log.Println("要求输入的是指针。")
		log.Println("程序退出。")
		return
	}
	var rpy replyMessage
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&rpy)
	if err != nil {
		log.Fatalln(err)
	}
	buf = bytes.NewBuffer(rpy.Data)
	resDec := gob.NewDecoder(buf)
	err = resDec.Decode(res)
	if err != nil {
		log.Println("在对输出的内容解码的时候发生了问题。")
		log.Println("这个问题可能和输入指针有点关系。")
		log.Println(err)
		return
	}

}
