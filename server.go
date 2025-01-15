package main

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

// read a WebSocket frame
func readWebSocketFrame(conn net.Conn) ([]byte, error) {
	// read the first two bytes (fin, opcode, mask bit, payload length)
	header := make([]byte, 2)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, err
	}

	// parse the header
	// fin := (header[0] & 0x80) != 0   // final frame flag
	opcode := header[0] & 0x0F          // opcode (text, binary, etc.)
	masked := (header[1] & 0x80) != 0   // fask flag (always true for client frames)
	payloadLen := int(header[1] & 0x7F) // payload length

	if payloadLen == 126 {
		// read next 2 bytes for extended payload length
		extended := make([]byte, 2)
		_, err = io.ReadFull(conn, extended)
		if err != nil {
			return nil, err
		}
		payloadLen = int(extended[0])<<8 | int(extended[1])
	}

	// read the masking key
	maskingKey := make([]byte, 4)
	if masked {
		_, err = io.ReadFull(conn, maskingKey)
		if err != nil {
			return nil, err
		}
	}

	// read the payload data
	payload := make([]byte, payloadLen)
	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return nil, err
	}

	// unmask the payload
	if masked {
		for i := 0; i < payloadLen; i++ {
			payload[i] ^= maskingKey[i%4]
		}
	}

	// handle opcodes like close and ping
	if opcode == 0x8 {
		conn.Close()
		return nil, fmt.Errorf("connection closed")
	}

	return payload, nil
}

// write a WebSocket frame
func writeWebSocketFrame(conn net.Conn, message []byte) error {
	// create a simple text frame
	frame := []byte{0x81} // FIN + Text frame opcode
	length := len(message)

	if length <= 125 {
		frame = append(frame, byte(length))
	} else if length <= 65535 {
		frame = append(frame, 126, byte(length>>8), byte(length&0xFF))
	} else {
		return fmt.Errorf("message too long")
	}

	frame = append(frame, message...)
	_, err := conn.Write(frame)
	return err
}

func main() {
	// start the server
	listener, err := net.Listen("tcp", ":8080")

	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()
	log.Println("Server started on :8080")

	// accept all incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Error accepting connection:", err)
			continue
		}

		// handle each connection in a goroutine
		go func() {
			// close the connection when the function returns
			defer conn.Close()

			buf := make([]byte, 1024)
			data, err := conn.Read(buf)
			if err != nil {
				return
			}

			// log the data
			// log.Println("Data received: ", string(buf[:data]))

			// parse the headers
			headers := make(map[string]string)
			lines := strings.Split(string(buf[:data]), "\r\n")

			for _, line := range lines {
				parts := strings.SplitN(line, ": ", 2)
				if len(parts) == 2 {
					headers[parts[0]] = parts[1]
				}
			}

			// verify the request
			if headers["Upgrade"] != "websocket" {
				log.Println("Connection is not a WebSocket connection")
				return
			}

			key := headers["Sec-WebSocket-Key"]
			if key == "" {
				conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n\r\n"))
				conn.Close()
				return
			}

			// create the hash key
			hash := sha1.New()
			hash.Write([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
			acceptKey := base64.StdEncoding.EncodeToString(hash.Sum(nil))

			// create the response
			response := fmt.Sprintf(
				"HTTP/1.1 101 Switching Protocols\r\n"+
					"Upgrade: websocket\r\n"+
					"Connection: Upgrade\r\n"+
					"Sec-WebSocket-Accept: %s\r\n\r\n",
				acceptKey,
			)

			// send the response
			_, err = conn.Write([]byte(response))

			if err != nil {
				log.Println("Error writing response:", err)
				conn.Close()
				return
			}

			log.Println("WebSocket connection established")

			// read and write frames for back and forth communication
			for {
				message, err := readWebSocketFrame(conn)

				// check if connection is disconnected or errored
				if err != nil {
					if err.Error() == "connection closed" {
						log.Println("Connection closed")
					} else {
						log.Println("Error reading frame:", err)
					}

					break
				}

				log.Println("Received:", string(message))

				// echo the message back
				err = writeWebSocketFrame(conn, []byte(string(message) + " (echo)"))

				if err != nil {
					log.Println("Error writing frame:", err)
					break
				}
			}
		}()
	}
}
