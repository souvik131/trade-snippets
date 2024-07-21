package ws

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	ConnMutex sync.RWMutex
	*url.URL
	*http.Header
	*websocket.Conn
	ReaderChannel chan *Reader
}

type MessageType int

type Reader struct {
	MessageType
	Message []byte
	Error   error
}

type Writer struct {
	MessageType
	Message []byte
}

const TEXT MessageType = websocket.TextMessage
const BINARY MessageType = websocket.BinaryMessage
const CLOSE MessageType = websocket.CloseMessage
const PING MessageType = websocket.PingMessage
const PONG MessageType = websocket.PongMessage

var initialized = false

func (c *Client) Connect(ctx *context.Context) ([]byte, error) {
	conn, response, err := websocket.DefaultDialer.DialContext(*ctx, c.URL.String(), *c.Header)
	c.ConnMutex.Lock()
	c.Conn = conn
	// c.CloseSignalChannel = make(chan struct{})
	c.ConnMutex.Unlock()
	if err != nil {
		return []byte{}, err
	}
	binaryResponse, err := io.ReadAll(response.Body)
	if err != nil {
		return []byte{}, err
	}
	if !initialized {
		c.ReaderChannel = make(chan *Reader, 100)
		initialized = true
	}
	return binaryResponse, nil
}

func (c *Client) Close(ctx *context.Context) error {

	c.ConnMutex.Lock()
	defer c.ConnMutex.Unlock()
	return c.Conn.Close()
}

func (c *Client) Read(ctx *context.Context) error {

	go func(c *Client) {
		for {
			messageType, message, err := c.Conn.ReadMessage()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				c.ReaderChannel <- &Reader{
					MessageType: MessageType(messageType),
					Message:     message,
					Error:       nil,
				}
				err := c.Close(ctx)
				if err != nil {
					log.Printf("websocket : failed closing connection -> %v", err)
				}
				return
			}
			c.ReaderChannel <- &Reader{
				MessageType: MessageType(messageType),
				Message:     message,
				Error:       nil,
			}
			if err != nil {

				log.Printf("websocket : reconnecting as error in connection during read -> %v", err)
				return
			}

		}
	}(c)
	return nil
}

func (c *Client) Write(ctx *context.Context, writer *Writer) error {
	c.ConnMutex.Lock()
	err := c.WriteMessage(int(writer.MessageType), writer.Message)
	c.ConnMutex.Unlock()

	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		err := c.Close(ctx)
		if err != nil {
			log.Printf("websocket : failed closing connection -> %v", err)
		}
		return nil
	}
	if err != nil {
		log.Printf("websocket : reconnecting as error in connection during write -> %v", err)
	}
	return err
}
