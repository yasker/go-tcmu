package comm

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/golang/protobuf/proto"

	"github.com/yasker/longhorn/block"
)

const (
	MESSAGE_LENGTH_SIZE = 4

	MSG_TYPE_READ_REQUEST   = 1
	MSG_TYPE_READ_RESPONSE  = 2
	MSG_TYPE_WRITE_REQUEST  = 3
	MSG_TYPE_WRITE_RESPONSE = 4
)

func EncodeLength(length uint32) []byte {
	bytes := make([]byte, MESSAGE_LENGTH_SIZE)
	binary.BigEndian.PutUint32(bytes, length)
	return bytes
}

func DecodeLength(bytes []byte) uint32 {
	return binary.BigEndian.Uint32(bytes)
}

func ReadRequest(conn io.Reader) (*block.Request, error) {
	lengthData := make([]byte, MESSAGE_LENGTH_SIZE)
	_, err := conn.Read(lengthData)
	if err == io.EOF {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("Fail to read message length size:", err)
	}

	length := DecodeLength(lengthData)
	data := make([]byte, length, length)
	if _, err := conn.Read(data); err != nil {
		return nil, fmt.Errorf("Fail to read message with size ", length, err)
	}

	req := &block.Request{}
	if err := proto.Unmarshal(data, req); err != nil {
		return nil, fmt.Errorf("Fail to decode message: ", err)
	}
	return req, nil
}

func SendResponse(conn io.Writer, resp *block.Response) error {
	data, err := proto.Marshal(resp)
	if err != nil {
		return fmt.Errorf("Fail to encode message: ", err)
	}
	length := len(data)
	lengthData := EncodeLength(uint32(length))
	if _, err := conn.Write(lengthData); err != nil {
		return fmt.Errorf("Fail to write message length size:", err)
	}
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("Fail to write message:", err)
	}
	return nil
}

func SendRequest(conn io.Writer, req *block.Request) error {
	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("Fail to encode message: ", err)
	}
	length := len(data)
	lengthData := EncodeLength(uint32(length))
	if _, err := conn.Write(lengthData); err != nil {
		return fmt.Errorf("Fail to write message length size:", err)
	}
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("Fail to write message:", err)
	}
	return nil
}

func ReadResponse(conn io.Reader) (*block.Response, error) {
	lengthData := make([]byte, MESSAGE_LENGTH_SIZE)
	_, err := conn.Read(lengthData)
	if err != nil {
		return nil, fmt.Errorf("Fail to read message length size:", err)
	}

	length := DecodeLength(lengthData)
	data := make([]byte, length, length)
	if _, err := conn.Read(data); err != nil {
		return nil, fmt.Errorf("Fail to read message with size ", length, err)
	}

	resp := &block.Response{}
	if err := proto.Unmarshal(data, resp); err != nil {
		return nil, fmt.Errorf("Fail to decode message: ", err)
	}
	return resp, nil
}

func SendData(conn io.Writer, buf []byte) error {
	_, err := conn.Write(buf)
	return err
}

func ReceiveData(conn io.Reader, buf []byte) error {
	_, err := conn.Read(buf)
	return err
}
