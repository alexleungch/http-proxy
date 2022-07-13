package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

var (
	version string
)

func ParseArgs() (string, string) {
	listenAddr := flag.String("l", ":8080", "listen address")
	forwardAddr := flag.String("f", "", "forwarding address")
	flagVersion := flag.Bool("v", false, "print version")

	flag.Parse()

	if *flagVersion {
		fmt.Println("version:", version)
		os.Exit(0)
	}

	if *forwardAddr == "" {
		flag.Usage()
		os.Exit(0)
	}

	return *listenAddr, *forwardAddr
}

func ListenAndServe(listenAddr string, forwardAddr string) {
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("listen addr %s failed: %s", listenAddr, err.Error())
	}

	log.Printf("accept %s to %s\n", listenAddr, forwardAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept %s error: %s\n", listenAddr, err.Error())
		}

		go HandleRequest(conn, forwardAddr)
	}
}

func HandleRequest(conn net.Conn, forwardAddr string) {
	d := net.Dialer{Timeout: time.Second * 10}

	proxy, err := d.Dial("tcp", forwardAddr)
	if err != nil {
		log.Printf("try connect %s -> %s failed: %s\n", conn.RemoteAddr(), forwardAddr, err.Error())
		conn.Close()
		return
	}
	log.Printf("connected: %s -> %s\n", conn.RemoteAddr(), forwardAddr)

	Pipe(conn, proxy)
}

func Pipe(src net.Conn, dest net.Conn) {
	var (
		readBytes  int64
		writeBytes int64
	)
	ts := time.Now()

	wg := sync.WaitGroup{}
	wg.Add(1)

	closeFun := func(err error) {
		dest.Close()
		src.Close()
	}

	go func() {
		defer wg.Done()
		n, err := io.Copy(dest, src)
		readBytes += n
		closeFun(err)
	}()

	n, err := io.Copy(src, dest)
	writeBytes += n
	closeFun(err)

	wg.Wait()
	log.Printf("connection %s -> %s closed: readBytes %d, writeBytes %d, duration %s", src.RemoteAddr(), dest.RemoteAddr(), readBytes, writeBytes, time.Now().Sub(ts))
}

func main() {
	listenAddr, forwardAddr := ParseArgs()
	ListenAndServe(listenAddr, forwardAddr)
}
