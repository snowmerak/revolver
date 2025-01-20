package main

import "net"

func GetFreeTcpPort() (int, error) {
	conn, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	return conn.Addr().(*net.TCPAddr).Port, nil
}
