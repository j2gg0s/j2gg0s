package main

import "net"

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			panic(err)
		}
		go func(conn net.Conn) {
			defer conn.Close()
			packet := make([]byte, 1024)
			for {
				n, err := conn.Read(packet)
				if err != nil {
					panic(err)
				}

				conn.Write(packet[:n])
			}
		}(conn)
	}
}
