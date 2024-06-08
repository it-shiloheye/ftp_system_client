package logging

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	initialiseclient "github.com/it-shiloheye/ftp_system/client/init_client"
	ftp_base "github.com/it-shiloheye/ftp_system_lib/base"
	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
	"github.com/it-shiloheye/ftp_system_lib/file_handler/v2"
)

type Loc string

var Logger = &LoggerStruct{

	comm:  make(chan *ftp_context.LogItem, 100),
	err_c: make(chan error, 100),
}
var ClientConfig = &initialiseclient.ClientConfigStruct{}
var lock = &sync.Mutex{}

type LoggerStruct struct {
	comm  chan *ftp_context.LogItem
	err_c chan error
}

var log_file = &filehandler.FileBasic{}
var log_err_file = &filehandler.FileBasic{}
var log_today_file = &filehandler.FileBasic{}

func init() {
	log.Println("loading logger")
	if len(ClientConfig.DataDir) < 1 {
		ClientConfig.DataDir = "./data"
	}

	loc := "ftp_system/client/main_thread/logging/logging_struct.go"
	log_file_p := ClientConfig.DataDir + "/log/log_file.txt"
	log_err_file_p := ClientConfig.DataDir + "/log/log_err_file.txt"
	log_today_file_p := ClientConfig.DataDir + "/log/sess/" + log_file_name() + ".txt"

	log.Printf("%s\nlog_file_p: %s\nlog_err_file_p: %s\n", loc, log_file_p, log_err_file_p)
	// os.Exit(1)
	var err1, err2, err3, err4 error

	err1 = os.MkdirAll(ClientConfig.DataDir+"/log/sess", fs.FileMode(ftp_base.S_IRWXO|ftp_base.S_IRWXU))
	if !errors.Is(err1, os.ErrExist) && err1 != nil {
		a := &ftp_context.LogItem{
			Location: loc,
			Time:     time.Now(),
			Message:  err1.Error(),
			Err:      true, CallStack: []error{err1},
		}
		log.Fatalln(a)
	}

	log_file.File, err2 = ftp_base.OpenFile(log_file_p, os.O_APPEND|os.O_RDWR|os.O_CREATE)
	if err2 != nil {
		b := &ftp_context.LogItem{
			Location: loc,
			Time:     time.Now(),
			Message:  err2.Error(),

			CallStack: []error{err2},
		}
		log.Fatalln(b)
	}

	log_err_file.File, err3 = ftp_base.OpenFile(log_err_file_p, os.O_APPEND|os.O_RDWR|os.O_CREATE)
	if err3 != nil {
		c := &ftp_context.LogItem{
			Location:  loc,
			Time:      time.Now(),
			Message:   err3.Error(),
			Err:       true,
			CallStack: []error{err3},
		}
		log.Fatalln(c)
	}

	log_today_file, err4 = filehandler.Create(log_today_file_p)
	if err4 != nil {
		c := &ftp_context.LogItem{
			Location:  loc,
			Time:      time.Now(),
			Message:   err4.Error(),
			Err:       true,
			CallStack: []error{err4},
		}
		log.Fatalln(c)
	}

	log.Println("successfull loaded logger")
}

func (ls *LoggerStruct) Log(li *ftp_context.LogItem) {

	ls.comm <- li
}

func (ls *LoggerStruct) Logf(loc Loc, str string, v ...any) {
	ls.comm <- &ftp_context.LogItem{
		Location: string(loc),
		Time:     time.Now(),
		Message:  fmt.Sprintf(str, v...),
	}
}

func (ls *LoggerStruct) LogErr(loc Loc, err error) {
	e := &ftp_context.LogItem{
		Location:  string(loc),
		Time:      time.Now(),
		Err:       true,
		Message:   err.Error(),
		CallStack: []error{err},
	}
	ls.Log(e)
}

func (ls *LoggerStruct) Engine(ctx ftp_context.Context) {
	lock.Lock()
	defer ctx.Finished()
	defer lock.Unlock()

	tc := time.NewTicker(time.Second)

	var li *ftp_context.LogItem

	queue := []*ftp_context.LogItem{}

	var txt, log_txt, err_txt string
	int_ := 0
	for ok := true; ok; {
		log_txt, err_txt = "", ""
		int_ += 1
		select {
		case _, ok = <-ctx.Done():

		case li = <-ls.comm:
			if li != nil {
				queue = append(queue, li)
			}
			// log.Println(int_, "none: new li")
			continue
		case <-tc.C:
		}

		for _, li := range queue {
			if li == nil {
				continue
			}

			txt = li.String() + "\n"
			if len(txt) < 2 {
				continue
			}
			log_txt = log_txt + txt
			if li.Err {
				err_txt = err_txt + txt
				log.SetOutput(os.Stderr)
				log.Print(txt)
			} else {
				log.SetOutput(os.Stdout)
				log.Printf("\n%s:\n%s", li.Location, li.Message)
			}

		}

		if len(log_txt) > 0 {
			log.SetOutput(log_today_file)
			log.Print(log_txt)

			log.SetOutput(log_file)
			log.Print(log_txt)
		}

		if len(err_txt) > 0 {
			log.SetOutput(log_err_file)
			log.Print(log_txt)

		}

		clear(queue)

		log.SetOutput(os.Stdout)
		// log.Println(int_, "none: done engine")
	}

}

func log_file_name() string {

	d := time.Now().Format(time.RFC3339)
	d1 := strings.ReplaceAll(d, " ", "_")
	d2 := strings.ReplaceAll(d1, ":", "")
	d3 := strings.Split(d2, "T")
	return d3[0]
}
