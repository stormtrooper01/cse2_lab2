package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
    	"errors"
    	"bytes"
    	"strconv"

	"github.com/stormtrooper01/cse2_lab2/httptools"
	"github.com/stormtrooper01/cse2_lab2/signal"
)

var (
	port = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", true, "whether to include tracing information into responses")
)

var (
	timeout = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}

	serversPoolTraffic = []int64{
		0,
		0,
		0,
	}
    
	serverStatus = []bool {
		true,
		true,
		true,
	}
)

func getTheBestServer() (string, error) {
	var theBestServerIndex = -1

	var minimalTraffic int64 = 9223372036854775807 //int64
	for i := 0; i < 3; i++ {
		if serversPoolTraffic[i] <= minimalTraffic && serverStatus[i] {
			minimalTraffic = serversPoolTraffic[i]
			theBestServerIndex = i
		}
	}
	if theBestServerIndex == -1 {
		return "", errors.New("every server is not healthy")
	}
	return serversPool[theBestServerIndex], nil
}

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string, index int) {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
            fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
		serverStatus[index] = false
		return
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	bytes := buf.Bytes()
	trafficString := string(bytes)
	traffic, err := strconv.ParseInt(trafficString, 10, 64)
	if err != nil {
		serverStatus[index] = false
		return
	}
	log.Printf("Response health %s: %d", dst, traffic)
	if resp.StatusCode != http.StatusOK {
		serverStatus[index] = false
		return
	}
	serverStatus[index] = true
	serversPoolTraffic[index] = traffic
}

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

func main() {
	flag.Parse()
	go func() {
		for range time.Tick(10 * time.Hour) {
			serversPoolTraffic[0] = 0
			serversPoolTraffic[1] = 0
			serversPoolTraffic[2] = 0
		}
	}()
    
	for index := 0; index < 3; index++ {
		server := serversPool[index]
		index := index
		go func() {
			for range time.Tick(10 * time.Second) {
				health(server, index)
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		var bestServer, err = getTheBestServer()
		if err == nil {
			log.Printf("Forwarding to server: %s", bestServer)
			forward(bestServer, rw, r)
		} else {
			log.Printf("Request error: %s", err.Error())
			rw.WriteHeader(http.StatusInternalServerError)
			rw.Write([]byte(err.Error()))
		}
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
