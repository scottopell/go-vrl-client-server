package main

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func ListenOnUDSSocket(path string) (*bufio.Reader, error) {
	if _, err := os.Stat(path); err == nil {
		if err := os.RemoveAll(path); err != nil {
			return nil, err
		}
	}

	listener, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}

	log.Printf("Waiting for connection on socket %s...", path)
	conn, err := listener.Accept()
	if err != nil {
		return nil, err
	}

	log.Print("Accepted connection from ", conn.RemoteAddr().Network())

	return bufio.NewReader(conn), nil
}

func ConnectToUDSSocket(path string, numRetries int) (io.Writer, error) {
	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return nil, err
	}

	var conn *net.UnixConn

	for attempts := 0; attempts <= numRetries; attempts++ {
		log.Print("Dialing specified addr:", addr)
		conn, err = net.DialUnix("unix", nil, addr)
		if err != nil {
			log.Println("Dial resulted in error: ", err)
		} else {
			break
		}
		time.Sleep(1 * time.Second)
	}

	if conn == nil {
		return nil, errors.New("retries exceeded, no dials to %s were successful")
	}
	log.Println("Connected to ", addr)

	return bufio.NewWriter(conn), nil
}
