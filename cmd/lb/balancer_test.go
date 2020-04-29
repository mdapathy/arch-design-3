package main

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"strings"
)

type TestCase struct {
	name           string
	serverConn     string
	serverHealth   string
	expectedServer string
}

var cases =[]TestCase {
	{"All servers working", "0 0 0", "1 1 1","1 2 3"},
	{"One server out-1", "0 0 0", "0 1 1","2 3"},
	{"One server out-2", "0 0 0", "1 0 1","1 3"},
	{"One server out-3", "0 0 0", "1 1 0","1 2"},
	{"Two servers out-1", "0 0 0", "1 0 0","1"},
	{"Two servers out-2", "0 0 0", "0 1 0","2"},
	{"Two servers out-3", "0 0 0", "0 0 1","3"},
	{"Three servers out", "0 0 0", "0 0 0",""},
	{"Least connections on server1", "5 25 50", "1 1 1","1"},
	{"Least connections on server2", "25 5 50", "1 1 1","2"},
	{"Least connections on server3", "50 25 5", "1 1 1","3"},
	{"Least connections on server that is out", "50 25 5", "1 1 0","2"},
}

func TestBalancer(t *testing.T) {
	for _, tcase := range cases {
		t.Run(tcase.name, func(t *testing.T) {
			serverConn := strings.Split(tcase.serverConn, " ")
			serverHealth := strings.Split(tcase.serverHealth, " ")
			safeServer = SafeServer{v: make(map[string]Server, len(serversPool))}
			for index, conn := range serverConn {
				connectionCount, _ := strconv.Atoi(conn)
				isHealthy, _ := strconv.ParseBool(serverHealth[index])
				safeServer.v[strconv.Itoa(index + 1)] =
					Server{connectionCount, isHealthy}
			}
			expectedServer := strings.Split(tcase.expectedServer, " ")
			assert.Contains(t, expectedServer, min(safeServer.v))
		})
	}
}

