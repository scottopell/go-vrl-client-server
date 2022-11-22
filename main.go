package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"time"
)

const (
	fakeLogLine = "Oct 17 14:33:33 | XSS | ERROR | (/viral/interactive/deliverables/holistic.go:3) | sed et dolorem minima et corrupti abcd veniam qui blanditiis optio explicabo et amet qui sint ut iure neque eveniet quod odio distinctio quas veniam voluptatibus quibusdam esse maiores dolores magni numquam sed deserunt quia odio fuga deserunt cumque a aliquam ad dolores dolore aut sapiente necessitatibus ut autem necessitatibus quam eveniet et omnis aut quos dolorem culpa nostrum quas provident tempora voluptate iure quos iste consequatur minima accusantium molestiae consequatur perspiciatis quis quia at incidunt non veritatis deserunt totam iure autem asperiores rerum officiis iusto et explicabo sunt et rerum molestiae hic dolore neque eum vel rerum perspiciatis autem et consequuntur consequatur aliquam dolore magni ea est illum accusamus rerum magnam neque odio voluptatibus est temporibus quo ullam nobis soluta quo ipsum temporibus perferendis et esse repellendus ea id explicabo nostrum repellat vero perferendis possimus optio consectetur deserunt aspern\n"
)

func listenForeverAndPrintThroughput(conn *net.UnixConn) {
	blackhole := BlackholeRecorder[[]byte]{}

	go func() {
		for {
			log.Println(blackhole.AvgThroughput())
			time.Sleep(3 * time.Second)
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

func startVectorWithConfig(config string) (*exec.Cmd, *bytes.Buffer, error) {
	tempConfigFile, err := os.CreateTemp("", "VectorDynamicConfig")
	if err != nil {
		return nil, nil, err
	}
	_, err = tempConfigFile.WriteString(config)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("Starting Vector with arg '-c %s'\n", tempConfigFile.Name())
	cmd := exec.Command("./vector/target/release/vector", "-c", tempConfigFile.Name())
	// TODO once I'm confident in the behavior of the vector subprocess, enable this
	// defer os.Remove(tempConfigFile.Name())

	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output

	err = cmd.Start()
	if err != nil {
		return nil, nil, err
	}

	return cmd, &output, nil
}

func PrintVectorOutputAndListenForDeath(cmd *exec.Cmd, stdoutAndStderrBytes *bytes.Buffer) {
	log.Printf("Vector has started succesfully, running in PID %d\n", cmd.Process.Pid)

	go func() {
		for {
			if stdoutAndStderrBytes.Len() > 0 {
				line, err := stdoutAndStderrBytes.ReadString('\n')
				if err != nil && err != io.EOF {
					log.Println("vector err:", err)
				} else {
					fmt.Println("\t", line)
				}
			}
			if cmd.ProcessState != nil {
				log.Println("Vector Has Exited! ProcessState: ", cmd.ProcessState)
				panic("Vector Quit.")
			}
		}
	}()

	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Panicln("Vector exited with non-nil error:", err)
		} else {
			log.Println("Vector exited normally.")
		}
	}()
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
	config := GenerateVectorConfig()

	// Listen on the vector output socket path and accept any connections
	go ListenOnUDSSocket(vectorOutputSocketPath, listenForeverAndPrintThroughput, errChan)

	log.Println("Going to start vector...")
	cmd, stdoutAndStderrBytes, err := startVectorWithConfig(config)
	if err != nil {
		log.Panicln(err)
	}
	defer cmd.Process.Kill()

	PrintVectorOutputAndListenForDeath(cmd, stdoutAndStderrBytes)

	// Connect to vector's input socket with 5 retries (timing)
	writer, err := ConnectToUDSSocket(vectorInputSocketPath, 5)
	if err != nil {
		log.Panicln(err)
	}

	go sendFakeLogDataForever(writer, errChan)

	anyErr := <-errChan
	if anyErr != nil {
		log.Panicln(anyErr)
	}
}
