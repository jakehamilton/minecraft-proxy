package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
)

func main() {
	listener, exception := net.Listen("tcp", ":25565")

	if exception != nil {
		fmt.Println(exception)
		return
	}

	for {
		connection, exception := listener.Accept()

		fmt.Println("Got Connection")

		if exception != nil {
			fmt.Println(exception)
			_ = listener.Close()
		} else {
			go handleHandshaking(connection)
		}
	}
}

func handleHandshaking(in net.Conn) {

	// Open our jsonFile
	jsonFile, err := os.Open("config.json")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	config := Config{}

	jsonBytes, exception := ioutil.ReadAll(jsonFile)

	if exception != nil {
		fmt.Println("Error reading config", exception)
		_ = in.Close()
		return
	}

	exception = json.Unmarshal(jsonBytes, &config)

	if exception != nil {
		fmt.Println("Error parsing config", exception)
		_ = in.Close()
		return
	}

	_ = jsonFile.Close()

	buf := make([]byte, 8192)
	for {
		// Read the incoming connection into the buffer.
		size, err := in.Read(buf)

		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading in:", err.Error())
				_ = in.Close()
				return
			}
		}

		if size == 0 {
			return
		}

		// Read the packet length
		_, cursor := ReadVarInt(buf)

		pos := cursor

		packetType, cursor := ReadVarInt(buf[pos:])
		pos += cursor

		if packetType != 0 {
			fmt.Printf("Packet was %x not Handshake\n", packetType)
			return
		}

		// Read the protocol version and skip over it
		protocolVersion, cursor := ReadVarInt(buf[pos:])

		fmt.Println("Protocol Version ", protocolVersion)

		pos += cursor

		serverAddress, cursor := ReadString(buf[pos:])

		fmt.Println("Server Address is ", serverAddress)

		targetHost, ok := config.Servers[serverAddress.(string)]

		if !ok {
			fmt.Println("Could not find host for ", serverAddress)
			_ = in.Close()
			return
		}

		fmt.Println("Target host is ", targetHost)
		// redirectMap[in.RemoteAddr().String()] = targetHost

		out, exception := net.Dial("tcp", targetHost)
		if exception != nil {
			fmt.Println("Could not connect to out ", serverAddress, exception)
			_ = in.Close()
			return
		}

		_, _ = out.Write(buf[:size])

		go handleIncoming(in, out)
		go handleOutgoing(in, out)
		break
	}
}

func handleIncoming(in net.Conn, out net.Conn) {
	buf := make([]byte, 8192)
	for {
		// Read the incoming connection into the buffer.
		size, err := in.Read(buf)

		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading in:", err.Error())
				_ = in.Close()
				_ = out.Close()
				return
			}
		}

		if size == 0 {
			return
		}

		_, err = out.Write(buf[:size])

		if err != nil && err != io.EOF {
			fmt.Println("Error writing out:", err.Error())
			_ = in.Close()
			_ = out.Close()
		}
	}
}

func handleOutgoing(in net.Conn, out net.Conn) {
	buf := make([]byte, 8192)
	for {
		// Read the incoming connection into the buffer.
		size, err := out.Read(buf)

		if err != nil {
			if err != io.EOF {
				fmt.Println("Error reading out:", err.Error())
				_ = in.Close()
				_ = out.Close()
				return
			}
		}

		if size == 0 {
			return
		}

		_, err = in.Write(buf[:size])

		if err != nil && err != io.EOF {
			fmt.Println("Error writing in:", err.Error())
			_ = in.Close()
			_ = out.Close()
		}
	}
}

func ReadVarInt(buf []byte) (interface{}, int) {
	numRead := 0
	result := 0

	var read byte

	for numRead, read = range buf {
		value := int32(read & 0b01111111)
		result |= int(value << (7 * numRead))

		//fmt.Println(hex.EncodeToString([]byte{read}))

		if numRead > 5 {
			fmt.Println("Numread overflow")
			return -1, numRead
		}
		if (read & 0b10000000) == 0 {
			break
		}
	}

	return result, numRead + 1
}

func ReadString(buf []byte) (interface{}, int) {
	stringLength, stringStart := ReadVarInt(buf)

	output := string(bytes.Runes(buf[stringStart:]))[:stringLength.(int)]

	return output, stringStart + len([]byte(output))

}
