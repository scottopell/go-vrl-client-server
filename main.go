package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"time"
)

const (
	fakeLogLine = "Oct 17 14:33:33 | XSS | ERROR | (/viral/interactive/deliverables/holistic.go:3) | sed et dolorem minima et corrupti abcd veniam qui blanditiis optio explicabo et amet qui sint ut iure neque eveniet quod odio distinctio quas veniam voluptatibus quibusdam esse maiores dolores magni numquam sed deserunt quia odio fuga deserunt cumque a aliquam ad dolores dolore aut sapiente necessitatibus ut autem necessitatibus quam eveniet et omnis aut quos dolorem culpa nostrum quas provident tempora voluptate iure quos iste consequatur minima accusantium molestiae consequatur perspiciatis quis quia at incidunt non veritatis deserunt totam iure autem asperiores rerum officiis iusto et explicabo sunt et rerum molestiae hic dolore neque eum vel rerum perspiciatis autem et consequuntur consequatur aliquam dolore magni ea est illum accusamus rerum magnam neque odio voluptatibus est temporibus quo ullam nobis soluta quo ipsum temporibus perferendis et esse repellendus ea id explicabo nostrum repellat vero perferendis possimus optio consectetur deserunt aspern\n"
)

func listenForeverAndPrintThroughput(conn *net.UnixConn) {
	blackhole := BlackholeRecorder[[]byte]{}
	connEmptyByteChecks := 0

	go func() {
		for {
			log.Println(conn, blackhole.AvgThroughput())
			if blackhole.totalBytes.Load() == 0 {
				connEmptyByteChecks++
			}
			if connEmptyByteChecks > 5 {
				log.Println("No data on this connection. Closing connection...")
				conn.Close()
				break
			}
			time.Sleep(1 * time.Second)
		}
	}()

	buf := make([]byte, 1024)

	for {
		nRead, _, err := conn.ReadFromUnix(buf)
		if nRead == 0 {
			// Nothing to read
		} else if err == io.EOF {
			log.Println("Closing connection, got EOF.")
			conn.Close()
		} else if err != nil {
			fmt.Println("Error while reading, error:", err)
			log.Panicln(err)
		} else {
			// Read something useful
			blackhole.Consume(buf)
		}
	}
}

func sendFakeLogDataForever(wr *bufio.Writer, errChan chan error) {
	log.Println("Sending Fake Log Data to input socket")
	data := []byte(fakeLogLine)
	dataLen := len(data)
	for {
		n, err := wr.Write(data)
		if err != nil {
			errChan <- err
		}
		if n != dataLen {
			errChan <- errors.New("did not successfully write all data to the socket")
		}
		err = wr.Flush()
		if err != nil {
			errChan <- err
		}
	}
}

func main() {
	errChan := make(chan error)
	shutdownChan := make(chan struct{})

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		close(shutdownChan)
	}()

	// Listen on the vector output socket path and accept any connections
	go ListenOnUDSSocket(vectorOutputSocketPath, listenForeverAndPrintThroughput, errChan)

	vectorRunner := NewVectorRunner(GenerateVectorConfig())
	err := vectorRunner.Start()
	if err != nil {
		log.Panicln(err)
	}
	defer vectorRunner.Stop()

	go vectorRunner.PrintOutputToStdout("\t")

	// Connect to vector's input socket with 5 retries (timing)
	writer, err := ConnectToUDSSocket(vectorInputSocketPath, 5)
	if err != nil {
		log.Panicln(err)
	}

	go sendFakeLogDataForever(writer, errChan)

	for {
		select {
		case <-shutdownChan:
			return
		case err = <-errChan:
			if err != nil {
				log.Panicln(err)
			}
		}
	}
}
