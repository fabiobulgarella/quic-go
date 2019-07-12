package main

import (
	"bytes"
	"crypto/tls"
	"flag"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/lucas-clemente/quic-go/internal/testdata"
	"github.com/lucas-clemente/quic-go/internal/utils"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"sync"
)

func main() {
	verbose := flag.Bool("v", false, "verbose")
	quiet := flag.Bool("q", false, "don't print the data")
	insecure := flag.Bool("insecure", false, "skip certificate verification")
	post := flag.Bool("p", false, "post data of specified dimension")
	flag.Parse()
	urls := flag.Args()

	logger := utils.DefaultLogger

	if *verbose {
		logger.SetLogLevel(utils.LogLevelDebug)
	} else {
		logger.SetLogLevel(utils.LogLevelInfo)
	}
	logger.SetLogTimeFormat("")

	roundTripper := &http3.RoundTripper{
		TLSClientConfig: &tls.Config{
			RootCAs:            testdata.GetRootCA(),
			InsecureSkipVerify: *insecure,
		},
	}
	defer roundTripper.Close()
	hclient := &http.Client{
		Transport: roundTripper,
	}

	if *post {
		addr := urls[0]
		dim, _ := strconv.Atoi(urls[1])
		logger.Infof("POST %d MB to %s", dim, addr)
		data := make([]byte, dim*1024*1024)
		rdata := bytes.NewReader(data)

		r, w := io.Pipe()
		m := multipart.NewWriter(w)
		go func() {
			defer w.Close()
			defer m.Close()
			part, err := m.CreateFormFile("uploadfile", "test.bin")
			if err != nil {
				return
			}
			if _, err = io.Copy(part, rdata); err != nil {
				return
			}
		}()
		rsp, err := hclient.Post(addr, m.FormDataContentType(), r)
		if err != nil {
			panic(err)
		}
		logger.Infof("Got response for %s: %#v", addr, rsp)

		body := &bytes.Buffer{}
		_, err = io.Copy(body, rsp.Body)
		if err != nil {
			panic(err)
		}
		if *quiet {
			logger.Infof("Request Body: %d bytes", body.Len())
		} else {
			logger.Infof("Request Body:")
			logger.Infof("%s", body.Bytes())
		}
	} else {
		var wg sync.WaitGroup
		wg.Add(len(urls))
		for _, addr := range urls {
			logger.Infof("GET %s", addr)
			go func(addr string) {
				rsp, err := hclient.Get(addr)
				if err != nil {
					panic(err)
				}
				logger.Infof("Got response for %s: %#v", addr, rsp)

				body := &bytes.Buffer{}
				_, err = io.Copy(body, rsp.Body)
				if err != nil {
					panic(err)
				}
				if *quiet {
					logger.Infof("Request Body: %d bytes", body.Len())
				} else {
					logger.Infof("Request Body:")
					logger.Infof("%s", body.Bytes())
				}
				wg.Done()
			}(addr)
		}
		wg.Wait()
	}
}
