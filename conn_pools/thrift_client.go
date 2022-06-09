package conn_pools

import (
	"time"

	"github.com/apache/thrift/lib/go/thrift"
)

type thriftClient struct {
	address string
	socket  *thrift.TSocket
	client  *thrift.TStandardClient
}

func newThriftClient(address string, timeout time.Duration) (*thriftClient, error) {
	socket, err := thrift.NewTSocketTimeout(address, timeout)
	if err != nil {
		return nil, err
	}

	//protocol
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()

	//transportFactory
	transportFactory := thrift.NewTFramedTransportFactory(thrift.NewTTransportFactory())

	//no buffered
	transport, err := transportFactory.GetTransport(socket)
	if err != nil {
		return nil, err
	}

	if err := transport.Open(); err != nil {
		return nil, err
	}

	iProto := protocolFactory.GetProtocol(transport)
	oProto := protocolFactory.GetProtocol(transport)

	var c thriftClient
	client := thrift.NewTStandardClient(iProto, oProto)
	c.client = client
	c.socket = socket
	c.address = address

	return &c, nil
}

func (c *thriftClient) close() error {
	err := c.socket.Close()
	if err != nil {
		return err
	}

	return nil
}
