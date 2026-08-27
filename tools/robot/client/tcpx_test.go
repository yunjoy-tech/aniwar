package client

import (
	"encoding/binary"
	"fmt"
	"gitee.com/aniwar2/aniwar/src/proto/pb"
	"gitee.com/aniwar2/musae/logger"
	"gitee.com/aniwar2/musae/tcpx"
	"net"
	"testing"
)

func Test_Pack(t *testing.T) {

	logger.Init("test")
	pack := tcpx.NewPackx(tcpx.ProtobufMarshaller{})
	req := &pb.C2LS_LoginReq{AccountId: "aaa", AccountPasswd: "111"}
	buf, err := pack.Pack(int32(pb.Protocols_PC2LS_LoginReq), 0, req)

	rsp := &pb.C2LS_LoginReq{}
	msg, err := pack.Unpack(buf, rsp)
	if err != nil {
		fmt.Print("ReqLogin Pack error:", err)
	}
	fmt.Println(msg)
}

func Test_BigEndian(t *testing.T) {
	var testInt int32 = 1002
	fmt.Printf("%d use big endian: \n", testInt)
	var testBytes []byte = make([]byte, 4)
	binary.BigEndian.PutUint32(testBytes, uint32(testInt))
	fmt.Println("int32 to bytes:", testBytes)

	convInt := int32(binary.LittleEndian.Uint32(testBytes))
	fmt.Printf("bytes to int32: %d\n\n", convInt)
}

func Test_3(t *testing.T) {
	var testInt int32 = 1002
	fmt.Printf("%d use big endian: \n", testInt)
	var testBytes []byte = make([]byte, 4)
	binary.BigEndian.PutUint32(testBytes, uint32(testInt))
	fmt.Println("int32 to bytes:", testBytes)

	convInt := int32(binary.LittleEndian.Uint32(testBytes))
	fmt.Printf("bytes to int32: %d\n\n", convInt)
}

func Test_Tcp(t *testing.T) {
	_, err := net.Dial("tcp", fmt.Sprintf("%s:29001", "http://192.168.1.44"))
	if err != nil {
		logger.Errorf(err.Error())
	}
}
