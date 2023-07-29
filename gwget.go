/*
 * a simple curl replacement
 * use only for download and get
 */
package main

import (
	"context"
	"fmt"
	"github.com/andelf/go-curl"
	"github.com/heetch/confita"
	confitaenv "github.com/heetch/confita/backend/env"
	confitafile "github.com/heetch/confita/backend/file"
	confitaflags "github.com/heetch/confita/backend/flags"
	"log"
	"os"
	"time"
)

var version string

type Config struct {
	Host      string `config:"gwghost,short=h,required,description=host url"`
	Format    string `config:"gwgformat,short=f,optional,description=accepted format"`
	Output    string `config:"gwgoutput,short=o,optional,description=output to file"`
	Redirect  bool   `config:"gwgredirect,short=r,optional,description=redirect enabled"`
	Timeout   int    `config:"gwgtimeout,short=t,optional,description=timeout ms"`
	Unsecure  bool   `config:"gwgunsecure,short=s,optional,description=accept unsecure tls"`
	Verbosity int    `config:"gwgverbosity,short=v,optional,description=log verbosity"`
}

// default values
var cfg = Config{
	Host:      "http://127.0.0.1",
	Format:    "html",
	Output:    "",
	Redirect:  false,
	Timeout:   100,
	Unsecure:  false,
	Verbosity: 0,
}

// input: easy curl instance; output: response code, relocation string
func rescodecheck(easy *curl.CURL) (int, string) {
	rescode := 0
	relocation := ""
	info, err := easy.Getinfo(curl.INFO_RESPONSE_CODE)
	if err != nil {
		log.Fatalln(err)
	}
	rescode = info.(int)

	// got relocation response
	if rescode == 302 {
		redirect, err := easy.Getinfo(curl.INFO_REDIRECT_URL)
		if err != nil {
			log.Fatalln(err)
		}
		relocation = redirect.(string)
	}
	return rescode, relocation
}

func writefile(ptr []byte, userdata interface{}) bool {
	file := userdata.(*os.File)
	if _, err := file.Write(ptr); err != nil {
		return false
	}
	return true
}

func getdata(ptr []byte, userdata interface{}) bool {
	ch, ok := userdata.(chan string)
	if ok {
		ch <- string(ptr)
		return true
	} else {
		log.Fatalln("ERROR getdata")
	}
	return false
}

var started = int64(0)

func showprogress(dltotal, dlnow, ultotal, ulnow float64, userdata interface{}) bool {

	if started == 0 {
		started = time.Now().Unix()
	}
	fmt.Printf("Downloaded: %3.2f%%, Speed: %.1fKiB/s \r", dlnow/dltotal*100, dlnow/1000/float64((time.Now().Unix()-started)))
	return true
}

// input: media type, file name; output: response code, relocation string
func httpget(httpurl string, media string, filename string) (int, string, string) {
	rescode := 0
	relocation := ""
	resdata := ""

	easy := curl.EasyInit()
	defer easy.Cleanup()

	if easy != nil {
		easy.Setopt(curl.OPT_URL, httpurl)

		easy.Setopt(curl.OPT_SSL_VERIFYPEER, cfg.Unsecure)
		easy.Setopt(curl.OPT_ENCODING, "Accept-Encoding: gzip, deflate")

		easy.Setopt(curl.OPT_HTTPHEADER, []string{"Accept: " + media})
		easy.Setopt(curl.OPT_USERAGENT, "")

		if cfg.Verbosity > 1 { // trace all
			easy.Setopt(curl.OPT_VERBOSE, true)
		}

		if filename != "" { // download file
			easy.Setopt(curl.OPT_WRITEFUNCTION, writefile)
			fp, _ := os.OpenFile(cfg.Output, os.O_WRONLY|os.O_CREATE, 0644)
			defer fp.Close() // defer close
			easy.Setopt(curl.OPT_WRITEDATA, fp)

			if cfg.Verbosity > 0 { // show progress bar
				easy.Setopt(curl.OPT_NOPROGRESS, false)
				easy.Setopt(curl.OPT_PROGRESSFUNCTION, showprogress)
			}

			if err := easy.Perform(); err != nil {
				log.Fatalln(err)
			}
			rescode, relocation = rescodecheck(easy)

		} else { // get http data

			easy.Setopt(curl.OPT_WRITEFUNCTION, getdata)
			// data reading process
			ch := make(chan string, 1024)
			go func(ch chan string) {
				for {
					resdata = <-ch
				}
			}(ch)
			easy.Setopt(curl.OPT_WRITEDATA, ch)

			if err := easy.Perform(); err != nil {
				log.Fatalln(err)
			}

			time.Sleep(time.Duration(cfg.Timeout) * time.Millisecond) // wait gorotine
			rescode, relocation = rescodecheck(easy)
		}
	} else {
		log.Fatalln("ERROR curl easy init")
	}
	return rescode, relocation, resdata
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
	var media string
	switch cfg.Format {
	case "json":
		media = "application/json"
	case "text":
		media = "text/plain"
	default:
		media = "text/html"
	}

	resdata := ""
	rescode, relocstr, resdata := httpget(cfg.Host, media, cfg.Output)
	// a maximum of two relocations
	if rescode == 302 {
		rescode, relocstr, resdata = httpget(relocstr, media, cfg.Output)
		if rescode == 302 {
			rescode, relocstr, resdata = httpget(relocstr, media, cfg.Output)
		}
	}
	if rescode == 200 && cfg.Output == "" {
		print(resdata)
	}
	if rescode != 200 {
		println("ERROR:", rescode)
	}
}
