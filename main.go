package main

/*
 * Copyright 2020 Peter Maguire
 * petermaguire.xyz
 */

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
	config, exception := readConfig()

	if exception != nil {
		fmt.Println("Error reading config ", exception)
		return
	}

	fmt.Println(os.Args)

	if len(os.Args) > 1 {
		editConfig(config)
		return
	}

	// Default to all interfaces port 25565
	if len(config.Listen) == 0 {
		config.Listen = ":25565"
	}

	fmt.Println(config.Servers)

	listener, exception := net.Listen("tcp", config.Listen)

	if exception != nil {
		fmt.Println(exception)
		return
	}

	fmt.Println("Listening on ", config.Listen)

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

func editConfig(config *Config) {
	if os.Args[1] == "add-server" {
		if len(os.Args) < 4 {
			fmt.Println("Usage: minecraft-proxy add-server hostname target:25565")
			return
		}
		host := os.Args[2]
		target := os.Args[3]

		config.Servers[host] = target

	} else if os.Args[1] == "del-server" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: minecraft-proxy del-server hostname")
			return
		}
		host := os.Args[2]
		delete(config.Servers, host)
	} else {
		fmt.Println("Usage: minecraft-proxy del-server or add-server")
		return
	}

	jsonOut, exception := json.MarshalIndent(&config, "", "\t")

	if exception != nil {
		fmt.Println("Error formatting JSON: ", exception)
		return
	}

	exception = ioutil.WriteFile("config.json", jsonOut, 0644)

	if exception != nil {
		fmt.Println("Error writing JSON: ", exception)
		return
	}
}

func readConfig() (*Config, error) {
	jsonFile, exception := os.Open("config.json")

	if exception != nil {
		return nil, exception
	}

	config := Config{}

	jsonBytes, exception := ioutil.ReadAll(jsonFile)

	if exception != nil {
		return nil, exception
	}

	exception = json.Unmarshal(jsonBytes, &config)

	if exception != nil {
		return nil, exception
	}

	_ = jsonFile.Close()

	return &config, nil
}

func handleHandshaking(in net.Conn) {

	config, exception := readConfig()

	if exception != nil {
		fmt.Println("Error reading config ", exception)
		_ = in.Close()
		return
	}

	buf := make([]byte, 8192)
	for {
		// Read the incoming connection into the buffer.
		size, err := in.Read(buf)

		if err != nil && err != io.EOF {
			fmt.Println("Error reading in:", err.Error())
			_ = in.Close()
			return
		}

		// Don't bother sending packets that are 0 length
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
			// Default host
			targetHost, ok = config.Servers["default"]
			if !ok {
				_ = in.Close()
				return
			}
		}

		fmt.Println("Target host is ", targetHost)

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
		size, exception := in.Read(buf)

		if exception != nil && exception != io.EOF {
			fmt.Println("Error reading in:", exception.Error())
			_ = in.Close()
			_ = out.Close()
			return
		}

		// Don't bother sending packets that are 0 length
		if size == 0 {
			return
		}

		_, exception = out.Write(buf[:size])

		if exception != nil && exception != io.EOF {
			fmt.Println("Error writing out:", exception.Error())
			_ = in.Close()
			_ = out.Close()
		}
	}
}

func handleOutgoing(in net.Conn, out net.Conn) {
	buf := make([]byte, 8192)
	for {
		// Read the incoming connection into the buffer.
		size, exception := out.Read(buf)

		if exception != nil && exception != io.EOF {
			fmt.Println("Error reading out:", exception.Error())
			_ = in.Close()
			_ = out.Close()
			return
		}

		// Don't bother sending packets that are 0 length
		if size == 0 {
			return
		}

		_, exception = in.Write(buf[:size])

		if exception != nil && exception != io.EOF {
			fmt.Println("Error writing in:", exception.Error())
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
