package mainthread

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	// "strings"

	"log"
	"time"

	initialiseclient "github.com/it-shiloheye/ftp_system/client/init_client"
	// "github.com/it-shiloheye/ftp_system/client/main_thread/actions"
	dir_handler "github.com/it-shiloheye/ftp_system/client/main_thread/dir_handler"
	"github.com/it-shiloheye/ftp_system/client/main_thread/logging"
	netclient "github.com/it-shiloheye/ftp_system/client/main_thread/network_client"

	ftp_base "github.com/it-shiloheye/ftp_system_lib/base"
	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
	filehandler "github.com/it-shiloheye/ftp_system_lib/file_handler/v2"
	// filehandler "github.com/it-shiloheye/ftp_system_lib/file_handler/v2"
	// filehandler "github.com/it-shiloheye/ftp_system_lib/file_handler/v2"
)

var ClientConfig = initialiseclient.ClientConfig
var Logger = logging.Logger
var FileTree = dir_handler.FileTree

func ticker(loc logging.Loc, i int) {

	// Logger.Logf(loc, "%d", i)
}

func MainThread(ctx ftp_context.Context) context.Context {
	loc := logging.Loc("MainThread(ctx ftp_context.Context) context.Context ")

	lock, ERR := dir_handler.Lock(ClientConfig.DataDir + "/index.lock")
	defer lock.Unlock()
	if ERR != nil {
		log.Println(ERR)
		log.Fatalln("cannot obtain lock on data/dir")
	}

	ticker(loc, 1)
	defer ctx.Wait()

	if len(ClientConfig.IncludeDir) < 1 {
		if len(ClientConfig.DirConfig.Path) > 0 {
			ClientConfig.IncludeDir = append(ClientConfig.IncludeDir, ClientConfig.DirConfig.Path)
		} else {
			log.Fatalln("add at least one file to include list or directory path")
		}
	}

	if ClientConfig.UpdateRate < 1 {
		ClientConfig.UpdateRate = time.Minute * 5
	} else {
		ClientConfig.UpdateRate = time.Duration(ClientConfig.UpdateRate)
	}

	tick := ClientConfig.UpdateRate

	client, err_ := netclient.NewNetworkClient(ctx)
	if err_ != nil {
		log.Fatalln(err_)
	}
	base_server := "https://127.0.0.1:8080"
	tyc := &TestServerConnection{
		tmp: map[string]string{},
		tc:  time.NewTicker(time.Second * 5),
	}

	test_server_connection(client, base_server, tyc)

	go UpdateFileTree(ctx.Add())

	bts := filehandler.NewBytesStore()
	buf := bytes.NewBuffer(make([]byte, 100_000))
	for ok := true; ok; {

		child_ctx := ctx.NewChild()
		child_ctx.SetDeadline(tick)
		Logger.Logf(loc, "starting client cycle")
		/**
		* five tasks:
		*	1. Read all files in directory
		*		- list all files (exclude .git) [done]
		*		- create a printout of list of files (current timestamped - incase of crash)
		* 	2. Check for any changes in directory compared to last scan
		*		- store past "ModTime" in special format
		*		- compare present and past mod-time for changes
		*	3. Add and commit all changes
		*   4. Hash all files in .git folder
		*		- read all files in .git
		*		- check for any changes in mod time (or new files)
		*		- hash where necessary
		*	5. Transmit over network any new changes where necessary
		 */

		err1 := dir_handler.ReadDir(child_ctx.Add(), ClientConfig.DirConfig)
		ticker(loc, 2)
		if err1 != nil {

			if ClientConfig.StopOnError {
				Logger.Logf(loc, "error occured, forced shutdown")
				log.Fatalln(err1.Error())
			}
			Logger.LogErr(loc, err1)
			continue
		}

		var d_c <-chan struct{}
	engine_loop:
		for _, file_ := range FileTree.FileState.Keys() {
			state, _ := FileTree.FileState.Get(file_)
			switch state {
			case dir_handler.FileStateToHash:
				d_c = HashingFunction(bts, file_)
			case dir_handler.FileStateToUpload:
				d_c = UploadingFunction(ctx, client, buf, file_)

			case dir_handler.FileStateToDownload:
				d_c = DownloadingFunction(client, file_)
			}
			select {
			case <-ctx.Done():
				break engine_loop
			case <-child_ctx.Done():
				break engine_loop
			case <-d_c:
			}
		}
		// child_ctx.Cancel()
		select {
		case _, ok = <-ctx.Done():

		case <-child_ctx.Done():

		}
		Logger.Logf(loc, "new tick")
	}

	return ctx
}

type TestServerConnection struct {
	tmp   map[string]string
	tc    *time.Ticker
	count int
}

func test_server_connection(client *http.Client, host string, tsc *TestServerConnection) {
	loc := logging.Loc(" test_server_connection(client *http.Client, host string, tsc *TestServerConnection)")
	Logger.Logf(loc, "test_server_connection")
	rc := netclient.Route{
		BaseUrl:  host,
		Pathname: "/ping",
	}
	_, err1 := netclient.MakeGetRequest(client, rc, &tsc.tmp)
	if err1 != nil {
		Logger.Logf(loc, "error here")
		tsc.count += 1
		if tsc.count < 5 {
			Logger.LogErr(loc, err1)

		} else {
			Logger.LogErr(loc, err1)
			os.Exit(1)
		}
		<-tsc.tc.C
		test_server_connection(client, host, tsc)
	}

	Logger.Logf(loc, "server connected successfully: %s", host)
}
func UpdateFileTree(ctx ftp_context.Context) {
	loc := logging.Loc("UpdateFileTree(ctx ftp_context.Context)")
	defer ctx.Finished()
	tc := time.NewTicker(time.Minute)
	for ok := true; ok; {
		select {
		case <-tc.C:
		case _, ok = <-ctx.Done():
		}

		err := dir_handler.WriteFileTree(ctx)
		if err != nil {
			Logger.LogErr(loc, err)
		}
		Logger.Logf(loc, "updated filetree successfully")
	}
}

func HashingFunction(bs *filehandler.BytesStore, file_p string) (d_c <-chan struct{}) {
	loc := logging.Loc("HashingFunction(bs *filehandler.BytesStore, file_p string) ")
	var err1, err2, err3 error
	fh, _ := FileTree.FileMap.Get(file_p)
	Logger.Logf(loc, "to hash: %s\nModTime: %s", file_p, fh.ModTime)
	bs.Reset()

	tmp_c := make(chan struct{}, 1)
	d_c = tmp_c
	defer close(tmp_c)
	if fh.File == nil {
		fh.File, err1 = ftp_base.OpenFile(file_p, os.O_RDONLY)
		if err1 != nil {
			Logger.LogErr(loc, &ftp_context.LogItem{
				Location:  string(loc),
				After:     fmt.Sprintf(`fh.File, err1 = ftp_base.OpenFile("%s",os.O_RDONLY)`, file_p),
				Message:   err1.Error(),
				CallStack: []error{err1},
			})

			return
		}
	}

	fh.Size, err2 = bs.ReadFrom(fh.File)
	if err2 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     `_, err1 = bs.ReadFrom(fh.File)`,
			Message:   err2.Error(),
			CallStack: []error{err2},
		})
		return
	}

	fh.Hash, err3 = bs.Hash()
	if err3 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     `fh.Hash, err2 = bs.Hash()`,
			Message:   err3.Error(),
			CallStack: []error{err3},
		})
		return
	}

	FileTree.FileState.Set(file_p, dir_handler.FileStateToUpload)
	Logger.Logf(loc, "done hashing: %s\nat: %s\nhash:\t%s", file_p, fmt.Sprint(time.Now()), fh.Hash)
	return
}

func UploadingFunction(ctx ftp_context.Context, client *http.Client, buf *bytes.Buffer, file_p string) (d_c <-chan struct{}) {
	loc := logging.Loc(`UploadingFunction(client *http.Client, file_p string)`)
	tmp_c := make(chan struct{}, 1)
	d_c = tmp_c
	defer close(tmp_c)

	client_id := ClientConfig.ClientId
	dir_id := ClientConfig.DirConfig.Id

	fh, _ := FileTree.FileMap.Get(file_p)
	route := ClientConfig.ServerIps[0] + "/upload/file/" + fh.Hash
	if fh.MetaData == nil {
		fh.MetaData = make(map[string]any)
	}
	fh.Set("client-id", client_id)
	fh.Set("dir-id", dir_id)

	data, err1 := json.Marshal(fh)
	if err1 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     `data, err1 := json.Marshal(fh)`,
			Message:   err1.Error(),
			CallStack: []error{err1},
		})
		return
	}

	buf.Reset()

	_, err2 := buf.Write(data)
	if err2 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     `_, err2 := buf.Write(data)`,
			Message:   err2.Error(),
			CallStack: []error{err2},
		})
		return
	}

	res, err3 := client.Post(route, "application/json", buf)
	if err3 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     fmt.Sprintf(`res, err3 := client.Post(%s,"application/json",buf)`, route),
			Message:   err3.Error(),
			CallStack: []error{err3},
		})
		return
	}

	buf.Reset()
	buf.ReadFrom(res.Body)
	resp := buf.Bytes()

	ts := map[string]any{}

	err4 := json.Unmarshal(resp, &ts)
	if err4 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     `err4 := json.Unmarshal(resp, &ts)`,
			Message:   err4.Error(),
			CallStack: []error{err4},
		})
		return
	}

	received, ok := ts["received"]
	if !ok {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location: string(loc),
			After:    `received, ok := ts["received"]`,
			Message:  `didn't receive a hash from server`,
		})
		return
	}

	if fh.Hash != received {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location: string(loc),
			After:    `fh.Hash != received`,
			Message:  fmt.Sprintf(`received the wrong hash from server:\nfile: %s\nsent: %s\n received: %s`, fh.Path, fh.Hash, received),
		})
		return
	}

	route = ClientConfig.ServerIps[0] + "/upload/stream/" + fh.Hash

	if fh.File == nil {
		fh.File, err4 = ftp_base.OpenFile(fh.Path, os.O_RDONLY)
		if err4 != nil {
			Logger.LogErr(loc, &ftp_context.LogItem{
				Location:  string(loc),
				After:     fmt.Sprintf(`fh.File, err4 =ftp_base.OpenFile(fh.Path:"%s",os.O_RDONLY)`, fh.Path),
				Message:   err4.Error(),
				CallStack: []error{err4},
			})
			return
		}
	}
	defer fh.Close()

	buf.Reset()
	_, err5 := buf.ReadFrom(fh.File)
	if err5 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     `_, err5 := buf.ReadFrom(fh.File)`,
			Message:   err5.Error(),
			CallStack: []error{err5},
		})
		return
	}

	tmp_2 := &map[string]any{
		"hash": fh.Hash,
		"data": buf.String(),
	}

	data, err6 := json.Marshal(tmp_2)
	if err6 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     `data, err6 := json.Marshal(tmp_2)`,
			Message:   err6.Error(),
			CallStack: []error{err6},
		})
		return
	}

	buf.Reset()
	buf.Read(data)
	log.Println(string(data))

	Logger.Logf(loc, "%s\nContent-Length: %d", route, fh.Size)
	req, err6 := http.NewRequestWithContext(ctx, "POST", route, buf)
	if err6 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     fmt.Sprintf(`res, err6 := http.NewRequestWithContext(ctx,"POST",%s,buf)`, route),
			Message:   err6.Error(),
			CallStack: []error{err6},
		})
		return
	}

	req.Header.Add("client-id", client_id)

	req.Header.Add("Content-type", "application/json")

	_, err7 := client.Post(route, "application/json", buf)
	if err7 != nil {
		log.Fatalln(err7)
	}

	Logger.Logf(loc, "done uploading: %s\nat: %s\response:\t%s", file_p, fmt.Sprint(time.Now()), string(resp))

	return
}

func DownloadingFunction(client *http.Client, file_p string) (d_c <-chan struct{}) {
	tmp_c := make(chan struct{}, 1)
	d_c = tmp_c
	defer close(tmp_c)

	return
}

/*


	res, err7 := client.Do(req)
	if err7 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     `res, err7 := client.Do(req)`,
			Message:   err7.Error(),
			CallStack: []error{err7},
		})
		return
	}

	Logger.Logf(loc, "%s\nSent: %d", route, fh.Path)
	buf.Reset()

*/
