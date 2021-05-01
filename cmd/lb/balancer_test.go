package main

import (
    "testing"
    
    . "gopkg.in/check.v1"	
)

func TestBalancer(t *testing.T) {
	assert := assert.New(t)
	serversPoolTraffic[0] = 100
	serversPoolTraffic[1] = 50
	serversPoolTraffic[2] = 200

	var bestServer1, _ = getTheBestServer()
	// Server without Error
	assert.Equal(serversPool[1], bestServer1)
	serverStatus[0] = false
	serverStatus[1] = false
	serverStatus[2] = false
	serversPoolTraffic[0] = -1
	serversPoolTraffic[1] = -1
	serversPoolTraffic[2] = -1
    
    var bestServer2, _ = getTheBestServer()
	// MinimalServer
	assert.Equal(serversPool[1], bestServer2)
	serverStatus[0] = false
	serversPoolTraffic[0] = 0
	serversPoolTraffic[1] = 200
	serverStatus[2] = false
	serversPoolTraffic[2] = 0

	var _, err3 = getTheBestServer()
	// Error is everywhere
	assert.Equal("every server is not healthy", err3.Error())
}
