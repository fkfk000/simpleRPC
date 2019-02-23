package main

import (
	"fmt"
	"log"
)

func main() {

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
	fmt.Println(*res)

}

type xxx struct {
	A, B int
	Name string
}
type Args struct {
	A, B int
}

func (x *xxx) Add(input Args) int {
	return input.A + input.B

}

type yyy struct {
	val int
}

func (y *yyy) Inc(val int) int {
	y.val = y.val + val
	return y.val
}

func (y *yyy) Set(val int) string {
	y.val = val
	return ""
}
