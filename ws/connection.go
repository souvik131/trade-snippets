package ws

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"

	"nhooyr.io/websocket"
)

type Client struct {
	ConnMutex sync.RWMutex
	*url.URL
	*http.Header
	*websocket.Conn
	ReaderChannel chan *Reader
	IsInitialized bool
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

const TEXT MessageType = 1
const BINARY MessageType = 2
const CLOSE MessageType = 9
const PING MessageType = 8
const PONG MessageType = 10

// var initialized = false

func (c *Client) Connect(ctx *context.Context) ([]byte, error) {

	conn, response, err := websocket.Dial(*ctx, c.URL.String(), &websocket.DialOptions{HTTPHeader: *c.Header})
	conn.SetReadLimit(4e6)
	// log.Println(c.URL.String(), response.Status)
	c.ConnMutex.Lock()
	c.Conn = conn
	c.ConnMutex.Unlock()
	if err != nil {
		return []byte{}, err
	}
	binaryResponse := []byte{}
	if response.Body != nil {
		binaryResponse, err = io.ReadAll(response.Body)
		if err != nil {
			return []byte{}, err
		}
	}
	if !c.IsInitialized {
		c.ReaderChannel = make(chan *Reader, 100)
		c.IsInitialized = true
	}
	return binaryResponse, nil
}

func (c *Client) Close(ctx *context.Context) error {

	c.ConnMutex.Lock()
	defer c.ConnMutex.Unlock()
	return c.Conn.Close(websocket.StatusNormalClosure, "ok")
}

func (c *Client) Read(ctx *context.Context) error {

	go func(c *Client) {
		for {
			messageType, message, err := c.Conn.Read(*ctx)
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
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
	err := c.Conn.Write(*ctx, websocket.MessageType(writer.MessageType), writer.Message)
	c.ConnMutex.Unlock()
	if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
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
