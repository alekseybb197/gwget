/*
 * a simple curl replacement
 * use only for download and get
 */
package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"github.com/heetch/confita"
	confitaenv "github.com/heetch/confita/backend/env"
	confitafile "github.com/heetch/confita/backend/file"
	confitaflags "github.com/heetch/confita/backend/flags"
	"github.com/schollz/progressbar/v3"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

var version string

type Config struct {
	Client    string `config:"gwgclient,short=c,optional,description=client string"`
	Host      string `config:"gwghost,short=h,required,description=host url"`
	Format    string `config:"gwgformat,short=f,optional,description=accepted format"`
	Output    string `config:"gwgoutput,short=o,optional,description=output to file"`
	Schema    string `config:"gwgschema,short=s,optional,description=schema http/https"`
	Timeout   int    `config:"gwgtimeout,short=t,optional,description=timeout ms"`
	Unsecure  bool   `config:"gwgunsecure,short=u,optional,description=accept unsecure tls"`
	Verbosity int    `config:"gwgverbosity,short=v,optional,description=log verbosity"`
}

// default values
var cfg = Config{
	Client:   "", // i.e. "curl/7.79.1"
	Host:     "http://127.0.0.1",
	Format:   "html",
	Output:   "",
	Schema:   "https", // default schema
	Timeout:  10000,
	Unsecure: false,
	// 0 - no log, 1 - progress bar, 2 - info, 3 - debug
	Verbosity: 1,
}

var webClient = http.Client{
	Timeout: time.Millisecond * time.Duration(cfg.Timeout),
}

var media string

func webrequest(url string) *http.Response {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header.Set("User-Agent", cfg.Client)
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept", media)

	if cfg.Verbosity > 2 {
		fmt.Printf("Request Headers: %+v\n", req.Header)
	}

	res, err := webClient.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	return res
}

func main() {
	// load actual values
	loader := confita.NewLoader(
		confitafile.NewOptionalBackend(".gwget.yaml"),
		confitaenv.NewBackend(),
		confitaflags.NewBackend(),
	)

	// process config error
	err := loader.Load(context.Background(), &cfg)
	if err != nil {
		log.Fatalln(err)
	}

	// set response accepted
	switch cfg.Format {
	case "json":
		media = "application/json"
	case "text":
		media = "text/plain"
	default:
		media = "text/html"
	}

	url := cfg.Host
	// fix schema if needed
	schema, _ := regexp.Compile(`^http.://`) // schema mask
	if !schema.MatchString(cfg.Host) {
		url = "https://" + cfg.Host
	}
	if cfg.Verbosity > 1 {
		fmt.Printf("Url: %+v\n", url)
	}

	if cfg.Output != "" {
		webClient.Timeout = -1 // disable timeout
	}

	res := webrequest(url)
	if res.Body == nil {
		log.Fatalln("Error: body is nil")
	}
	defer res.Body.Close()

	if cfg.Verbosity > 2 {
		fmt.Printf("Response StatusCode %+v\n", res.StatusCode)
		fmt.Printf("Response Header %+v\n", res.Header)
	}

	// catch fail code
	if res.StatusCode != 200 && res.StatusCode != 400 && res.StatusCode != 404 && res.StatusCode != 401 {
		log.Fatalf("failed to fetch data: %s", res.Status)
	}

	// choose properly context reader
	var reader io.ReadCloser
	switch res.Header.Get("Content-Encoding") {
	case "gzip":
		reader, err = gzip.NewReader(res.Body)
		defer reader.Close()
	default:
		reader = res.Body
	}

	if cfg.Output == "" {
		// fetch web content

		bytesbody, readErr := io.ReadAll(reader)
		if readErr != nil {
			log.Fatalln(readErr)
		}
		fmt.Print(string(bytesbody))

	} else {
		// save to file
		file, err := os.Create(cfg.Output)
		if err != nil {
			log.Fatalln(err)
		}
		defer file.Close()

		var size int64
		if cfg.Verbosity > 0 {
			bar := progressbar.DefaultBytes(
				res.ContentLength,
				"downloading",
			)
			size, err = io.Copy(io.MultiWriter(file, bar), reader)
		} else {
			size, err = io.Copy(file, reader)
		}
		if err != nil {
			log.Fatalln(err)
		}
		if cfg.Verbosity > 1 {
			fmt.Printf("File %s with %d bytes downloaded.\n", cfg.Output, size)
		}
	}

}
