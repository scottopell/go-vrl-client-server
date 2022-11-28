package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func ListenOnUDSSocket(path string, handler func(*net.UnixConn), errChan chan error) {
	if _, err := os.Stat(path); err == nil {
		if err := os.RemoveAll(path); err != nil {
			errChan <- err
			return
		}
	}

	unixAddr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		errChan <- err
		return
	}

	listener, err := net.ListenUnix("unix", unixAddr)
	if err != nil {
		errChan <- err
		return
	}

	for {
		log.Printf("Waiting for connection on socket %q...", path)
		conn, err := listener.AcceptUnix()
		if err != nil {
			errChan <- err
			return
		}
		log.Printf("Accepted connection on socket %q from %q. Starting handler.", path, conn.RemoteAddr().Network())
		go handler(conn)
	}
}

func ConnectToUDSSocket(path string, numRetries int) (*bufio.Writer, error) {
	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return nil, err
	}

	var conn *net.UnixConn

	for attempts := 0; attempts <= numRetries; attempts++ {
		log.Print("Dialing specified addr:", addr)
		conn, err = net.DialUnix("unix", nil, addr)
		if err != nil {
			log.Println("Dial resulted in error:", err)
		} else {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if conn == nil {
		return nil, fmt.Errorf("retries exceeded, no dials to %s were successful", path)
	}
	log.Println("Dial connected to", addr)

	return bufio.NewWriter(conn), nil
}
