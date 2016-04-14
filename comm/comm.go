package comm

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/gogo/protobuf/proto"

	"github.com/yasker/longhorn/block"
)

const (
	MSG_HEADER_LENGTH = 5

	MSG_TYPE_READ_REQUEST   = 1
	MSG_TYPE_READ_RESPONSE  = 2
	MSG_TYPE_WRITE_REQUEST  = 3
	MSG_TYPE_WRITE_RESPONSE = 4
)

func EncodeLength(length uint32) []byte {
	bytes := make([]byte, MSG_HEADER_LENGTH)
	binary.BigEndian.PutUint32(bytes, length)
	return bytes
}

func DecodeLength(bytes []byte) uint32 {
	return binary.BigEndian.Uint32(bytes)
}

func SendRequest(conn io.Writer, req *block.Request) error {
	data, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("Fail to encode message: ", err)
	}
	return send(conn, data)
}

func SendResponse(conn io.Writer, resp *block.Response) error {
	data, err := proto.Marshal(resp)
	if err != nil {
		return fmt.Errorf("Fail to encode message: ", err)
	}
	return send(conn, data)
}

func send(conn io.Writer, data []byte) error {
	length := len(data)
	if length >= (1 << (MSG_HEADER_LENGTH * 8)) {
		return fmt.Errorf("Length exceed maximum header length: ", length)
	}
	lengthData := EncodeLength(uint32(length))
	if _, err := conn.Write(lengthData); err != nil {
		return fmt.Errorf("Fail to write message length size:", err)
	}
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("Fail to write message:", err)
	}
	return nil
}

func ReadRequest(conn io.Reader) (*block.Request, error) {
	data, err := receive(conn)
	if err != nil {
		return nil, err
	}

	req := &block.Request{}
	if err := proto.Unmarshal(data, req); err != nil {
		return nil, fmt.Errorf("Fail to decode message: ", err)
	}
	return req, nil
}

func ReadResponse(conn io.Reader) (*block.Response, error) {
	data, err := receive(conn)
	if err != nil {
		return nil, err
	}

	resp := &block.Response{}
	if err := proto.Unmarshal(data, resp); err != nil {
		return nil, fmt.Errorf("Fail to decode message: ", err)
	}
	return resp, nil
}

func receive(conn io.Reader) ([]byte, error) {
	lengthData := make([]byte, MSG_HEADER_LENGTH)
	_, err := io.ReadFull(conn, lengthData)
	if IsEOF(err) {
		return nil, io.EOF
	}
	if err != nil {
		return nil, fmt.Errorf("Fail to read message length size:", err)
	}

	length := DecodeLength(lengthData)
	if length == 0 {
		return nil, fmt.Errorf("Fail to decode message length size")
	}
	data := make([]byte, length)
	if _, err := io.ReadFull(conn, data); err != nil {
		return nil, fmt.Errorf("Fail to read message with size ", length, err)
	}
	return data, nil
}

func SendData(conn io.Writer, buf []byte) error {
	_, err := conn.Write(buf)
	return err
}

func ReceiveData(conn io.Reader, buf []byte) error {
	_, err := io.ReadFull(conn, buf)
	return err
}

func IsEOF(err error) bool {
	if err == nil {
		return false
	} else if err == io.EOF {
		return true
	} else if oerr, ok := err.(*net.OpError); ok {
		if strings.HasSuffix(oerr.Err.Error(), "use of closed network connection") {
			return true
		}
	}
	return false
}
