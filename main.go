package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

const helpText = `
http-proxy is a tool to proxy http https request.

Usage:
  http-proxy --port=8080
`

func usage() {
	exitCode := -1
	fmt.Fprintf(os.Stderr, "%v\n", helpText)
	os.Exit(exitCode)
}

var port string
var logger = log.New(os.Stdout, "http-proxy: ", log.LstdFlags)

func init() {
	flag.StringVar(&port, "port", "8080", "server listen port")
	flag.Parse()
}

func exit(code int) {
	os.Exit(code)
}
func startServer() error {
	logger.Printf("start server listen port=%s\n", port)
	listen, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Printf("start server err=%s\n", err.Error())
		exit(-1)
	}
	for {
		conn, err := listen.Accept()
		if err != nil {
			log.Printf("listen accept err=%s\n", err.Error())
			exit(-1)
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("read from requsest err=%s close connection", err.Error())
		return
	}
	buf = buf[:n]
	bb := bytes.NewBuffer(buf)
	//ignore Proxy-Connection: Keep-Alive
	//example GET http://sunmi.com/ HTTP/1.1
	first, err := bb.ReadBytes('\n')
	if err != nil {
		log.Printf("decode http request fisrt line %s", string(first))
		return
	}
	method := string(first[:bytes.Index(first, []byte{' '})])
	//example Host: www.baidu.com\r\n
	second, err := bb.ReadBytes('\n')
	if err != nil {
		log.Printf("decode http request second line %s", string(second))
		return
	}
	host := string((second[bytes.Index(second, []byte{' '})+1 : len(second)-2]))
	isTunnel := method == http.MethodConnect
	if !strings.Contains(host, ":") {
		if !isTunnel {
			host += ":80"
		} else {
			host += ":443"
		}
	}
	target, err := net.Dial("tcp", host)
	if err != nil {
		log.Printf("net dial target=%s err=%s\n", host, err.Error())
		return
	}
	if isTunnel {
		resp := []byte(`HTTP/1.1 200 OK`)
		resp = append(resp, []byte{'\r', '\n', '\r', '\n'}...)
		conn.Write(resp)
	} else {
		target.Write(buf)
	}
	go io.Copy(target, conn)
	_, err = io.Copy(conn, target)
	if err != nil {
		log.Printf("io copy err=%s", err.Error())
	}
}

func main() {
	startServer()
}
