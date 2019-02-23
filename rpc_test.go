package main

import (
	"fmt"
	"log"
	"sync"
	"testing"
)

func TestBasicFunction(t *testing.T) {
	args := Args{1, 2}
	end := Client{Adress: "http://localhost:9090/v1/api"}
	resBuf, err := end.SendRequest("xxx.Add", args)
	if err != nil {
		log.Fatalln(err)
	}
	ok := ResultIsOk(resBuf)
	fmt.Println(ok)
	res := new(int)
	ConvertToNormalType(resBuf, res)
	if ok != true {
		t.Error("结果有问题")
	}
	if *res != 3 {
		t.Error("结果有问题")
	}
}

func TestTwoFunctions(t *testing.T) {
	args := Args{1, 2}
	end := Client{Adress: "http://localhost:9090/v1/api"}
	resBuf, err := end.SendRequest("xxx.Add", args)
	if err != nil {
		log.Fatalln(err)
	}
	ok := ResultIsOk(resBuf)
	fmt.Println(ok)
	res := new(int)
	ConvertToNormalType(resBuf, res)
	if ok != true {
		t.Error("结果有问题")
	}
	if *res != 3 {
		t.Error("结果有问题")
	}
	resBuf, err = end.SendRequest("yyy.Set", 0)
	ok = ResultIsOk(resBuf)
	if ok != true {
		t.Error("第二个函数第一个OK的结果有问题")
	}
	var resString string
	ConvertToNormalType(resBuf, &resString)
	if resString != "" {
		log.Println(resString)
		t.Error("第二个返回的第一个buf不对")
	}
	resBuf, err = end.SendRequest("yyy.Inc", 1)
	if err != nil {
		log.Fatalln(err)
	}
	ok = ResultIsOk(resBuf)
	if ok != true {
		t.Error("第二个函数OK的结果有问题")
	}
	res2 := new(int)
	ConvertToNormalType(resBuf, res2)
	if *res2 != 1 {
		log.Println(*res2)
		t.Error("第二个返回的buf不对")
	}

}

func TestMultiThread(t *testing.T) {
	wg := new(sync.WaitGroup)
	wg.Add(1000)
	end := Client{Adress: "http://localhost:9090/v1/api"}
	end.SendRequest("yyy.Set", 0)
	for i := 0; i < 1000; i++ {
		go parallel(&end, wg)
	}
	wg.Wait()
	resBuf, err := end.SendRequest("yyy.Inc", 0)
	if err != nil {
		log.Fatalln(err)
	}
	ok := ResultIsOk(resBuf)
	if ok != true {
		log.Fatalln("无法返回正确的值")
	}
	res := new(int)
	ConvertToNormalType(resBuf, res)
	if *res != 1000 {
		t.Error("没有返回预期的值")
	}
}

func parallel(end *Client, wg *sync.WaitGroup) {
	resBuf, err := end.SendRequest("yyy.Inc", 1)
	if err != nil {
		log.Fatalln(err)
	}
	ok := ResultIsOk(resBuf)
	if ok != true {
		log.Fatalln("无法返回正确的值")
	}
	res := new(int)
	ConvertToNormalType(resBuf, res)
	wg.Done()
}
