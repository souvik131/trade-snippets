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

const maxMessageBuffer = 10000 // Increased buffer size

func (c *Client) Connect(ctx *context.Context) ([]byte, error) {
	log.Printf("Connecting to WebSocket: %s", c.URL.String())

	options := &websocket.DialOptions{
		HTTPHeader: *c.Header,
	}

	conn, response, err := websocket.Dial(*ctx, c.URL.String(), options)
	if err != nil {
		log.Printf("WebSocket connection error: %v", err)
		return []byte{}, err
	}

	c.ConnMutex.Lock()
	c.Conn = conn
	c.ConnMutex.Unlock()

	binaryResponse := []byte{}
	if response.Body != nil {
		binaryResponse, err = io.ReadAll(response.Body)
		if err != nil {
			log.Printf("Error reading response body: %v", err)
			return []byte{}, err
		}
	}

	if !c.IsInitialized {
		c.ReaderChannel = make(chan *Reader, maxMessageBuffer)
		c.IsInitialized = true
		// log.Println("WebSocket client initialized with buffer size:", maxMessageBuffer)
	}

	// log.Println("WebSocket connection established")
	return binaryResponse, nil
}

func (c *Client) Close(ctx *context.Context) error {
	// log.Println("Closing WebSocket connection")
	c.ConnMutex.Lock()
	defer c.ConnMutex.Unlock()

	if c.Conn != nil {
		return c.Conn.Close(websocket.StatusNormalClosure, "ok")
	}
	return nil
}

func (c *Client) Read(ctx *context.Context) error {
	// log.Println("Starting WebSocket read loop")

	go func(c *Client) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in read loop: %v", r)
			}
		}()

		for {
			select {
			case <-(*ctx).Done():
				// log.Println("Context cancelled, stopping read loop")
				return
			default:
				c.ConnMutex.RLock()
				if c.Conn == nil {
					c.ConnMutex.RUnlock()
					// log.Println("Connection is nil, stopping read loop")
					return
				}

				messageType, message, err := c.Conn.Read(*ctx)
				c.ConnMutex.RUnlock()

				if err != nil {
					if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
						// log.Println("WebSocket closed normally")
						return
					}

					// Check if channel is full
					select {
					case c.ReaderChannel <- &Reader{
						MessageType: MessageType(messageType),
						Message:     nil,
						Error:       err,
					}:
					default:
						// log.Println("Reader channel full, dropping error message")
					}

					log.Printf("WebSocket read error: %v", err)
					return
				}

				// Try to send message with non-blocking select
				select {
				case c.ReaderChannel <- &Reader{
					MessageType: MessageType(messageType),
					Message:     message,
					Error:       nil,
				}:
				default:
					log.Printf("Reader channel full, dropping market data message (buffer: %d)", len(c.ReaderChannel))
				}
			}
		}
	}(c)

	return nil
}

func (c *Client) Write(ctx *context.Context, writer *Writer) error {
	c.ConnMutex.Lock()
	defer c.ConnMutex.Unlock()

	if c.Conn == nil {
		return nil
	}

	err := c.Conn.Write(*ctx, websocket.MessageType(writer.MessageType), writer.Message)
	if err != nil {
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			// log.Println("WebSocket closed during write")
			return c.Close(ctx)
		}
		log.Printf("WebSocket write error: %v", err)
		return err
	}

	return nil
}
