package integration

import (
	"log"
	"net/http"
	"testing"
	"time"
	"fmt"
    
    . "gopkg.in/check.v1"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

var (
	m = map[string]int {
		"server1:8080": 0,
		"server2:8080": 0,
		"server3:8080": 0,
	}
	avgTime = [10]float64{}
)

func requestSender(t *testing.T, finished chan bool, serverCounter map[string]int) {
	assert := assert.New(t)
	counter := 0
	for range time.Tick(11 * time.Second) {
		start := time.Now()
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			t.Error(err)
			finished <- true
			return
		}
		duration := time.Since(start).Seconds()
		avgTime[counter] = duration
		serverCounter[resp.Header.Get("lb-from")] += 1
		t.Logf("response from [%s]", resp.Header.Get("lb-from"))
		counter += 1
		if counter == 10 {
			break
		}
	}
	log.Println(serverCounter["server1:8080"])
	assert.True(serverCounter["server1:8080"] >= 3)
	log.Println(serverCounter["server2:8080"])
	assert.True(serverCounter["server2:8080"] >= 3)
	log.Println(serverCounter["server3:8080"])
	assert.True(serverCounter["server3:8080"] >= 3)
	var avg float64 = 0
	for i := 0; i < 10; i++ {
		avg += avgTime[i]
	}
	avg /= 10
	assert.True(avg < client.Timeout.Seconds())
	log.Println("Benchmark is ok")
	log.Printf("Benchmark avg: %g", avg)
	finished <- true
}

func TestBalancer(t *testing.T) {
	finished := make(chan bool)
	go requestSender(t, finished, m)
	<- finished
	log.Println("tests are ok!")
}
