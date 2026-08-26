package logic

import (
	"encoding/binary"
	"fmt"
	"errors"
	"github.com/yunjoy-tech/musae/logger"
	"github.com/yunjoy-tech/musae/tcpx"
	"net"
	"testing"
	"time"
)

func PackWithMarshaller(message tcpx.Message, marshaller tcpx.Marshaller) ([]byte, error) {
	if marshaller == nil {
		marshaller = tcpx.JsonMarshaller{}
	}
	var e error
	var lengthBuf = make([]byte, 4)
	var messageIDBuf = make([]byte, 4)
	binary.BigEndian.PutUint32(messageIDBuf, uint32(message.MessageID))
	var headerLengthBuf = make([]byte, 4)
	var bodyLengthBuf = make([]byte, 4)
	var headerBuf []byte
	var bodyBuf []byte
	// headerBuf, e = json.Marshal(message.Header)
	// if e != nil {
	//	return nil, e
	// }
	binary.BigEndian.PutUint32(headerLengthBuf, uint32(len(headerBuf)))
	if message.Body != nil {
		bodyBuf, e = marshaller.Marshal(message.Body)
		if e != nil {
			return nil, e
		}
	}

	binary.BigEndian.PutUint32(bodyLengthBuf, uint32(len(bodyBuf)))
	var content = make([]byte, 0, 1024)

	content = append(content, messageIDBuf...)
	content = append(content, headerLengthBuf...)
	content = append(content, bodyLengthBuf...)
	content = append(content, headerBuf...)
	content = append(content, bodyBuf...)

	binary.BigEndian.PutUint32(lengthBuf, uint32(len(content)))

	var packet = make([]byte, 0, 1024)

	packet = append(packet, lengthBuf...)
	packet = append(packet, content...)
	return packet, nil
}

func TestHeartbeat(t *testing.T) {

	logger.Init("log", "test")
	var serverStart = make(chan int, 1)
	var testResult = make(chan error, 1)
	go func() {
		time.Sleep(40 * time.Second)
		testResult <- nil
	}()

	// client
	go func() {
		<-serverStart

		conn, e := net.Dial("tcp", "localhost:7008")

		if e != nil {
			testResult <- fmt.Errorf(": %w", e)
			panic(any(e))
		}
		var heartBeat []byte
		heartBeat, e = PackWithMarshaller(tcpx.Message{
			MessageID: tcpx.DEFAULT_HEARTBEAT_MESSAGEID,
			// Header:    nil,
			Body: nil,
		}, nil)
		if e != nil {
			testResult <- fmt.Errorf(": %w", e)
			panic(e)
		}

		fmt.Println(heartBeat)
		for {
			_, e = conn.Write(heartBeat)
			if e != nil {
				fmt.Println(e.Error())
				testResult <- fmt.Errorf(": %w", e)
				break
			}
			time.Sleep(5 * time.Second)
		}
	}()

	// server
	go func() {
		go func() {
			time.Sleep(time.Second * 10)
			serverStart <- 1
		}()

		srv := tcpx.NewTcpX(nil)

		srv.HeartBeatModeDetail(true, 5*time.Second, false, tcpx.DEFAULT_HEARTBEAT_MESSAGEID)

		// srv.RewriteHeartBeatHandler(1300, func(c *tcpx.Context) {
		//	fmt.Println("rewrite heartbeat handler")
		//	c.RecvHeartBeat()
		// })

		srv.OnClose = OnClose
		srv.OnConnect = OnConnect

		tcpx.SetLogMode(tcpx.DEBUG)

		err := srv.ListenAndServe("tcp", ":7008")
		if err != nil {
			fmt.Println(err)
			t.Fail()
		}
	}()

	e := <-testResult
	if e != nil {
		fmt.Println(e.Error())
		t.Fail()
	}
}

func OnConnect(c *tcpx.Context) {
	fmt.Printf("connecting from remote host %s network %s", c.ClientIP(), c.Network())
}
func OnClose(c *tcpx.Context) {
	fmt.Printf("connecting from remote host %s network %s has stoped", c.ClientIP(), c.Network())
}

func RunClient(srvAddr string, testResult chan error) {
	// client -> gate
	go func() {

		conn, e := net.Dial("tcp", srvAddr)

		if e != nil {
			testResult <- fmt.Errorf(": %w", e)
			panic(any(e))
		}
		var heartBeat []byte
		heartBeat, e = PackWithMarshaller(tcpx.Message{
			MessageID: tcpx.DEFAULT_HEARTBEAT_MESSAGEID,
			// Header:    nil,
			Body: nil,
		}, nil)
		if e != nil {
			testResult <- fmt.Errorf(": %w", e)
			panic(e)
		}

		for {
			_, e = conn.Write(heartBeat)
			if e != nil {
				fmt.Println(e.Error())
				testResult <- fmt.Errorf(": %w", e)
				break
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

// 测试gate login
func TestGate(t *testing.T) {
	var testResult = make(chan error, 1)
	go func() {
		time.Sleep(40 * time.Second)
		testResult <- nil
	}()

	for i := 0; i <= 1; i++ {
		RunClient("localhost:13001", testResult)
		// RunClient("localhost:12002", testResult)
	}

	e := <-testResult
	if e != nil {
		fmt.Println(e.Error())
		t.Fail()
	}
}
