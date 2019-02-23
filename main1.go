package main

import (
	"net/http"
)

func main() {
	newServer := MakeNewServer()
	xx := &xxx{1, 2, "for test"}
	yy := &yyy{0}
	newServer.Install(xx)
	newServer.Install(yy)
	http.Handle("/v1/api", newServer)
	http.ListenAndServe(":9090", nil)
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
