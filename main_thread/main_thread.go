package mainthread

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	// "strings"

	"log"
	"time"

	initialiseclient "github.com/it-shiloheye/ftp_system_client/init_client"
	// "github.com/it-shiloheye/ftp_system_client/main_thread/actions"
	dir_handler "github.com/it-shiloheye/ftp_system_client/main_thread/dir_handler"
	"github.com/it-shiloheye/ftp_system_client/main_thread/logging"
	netclient "github.com/it-shiloheye/ftp_system_client/main_thread/network_client"

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

	go dir_handler.UpdateFileTree(ctx.Add(), ClientConfig.DataDir+"/file-tree.lock", ClientConfig.DataDir+"/file-tree.json")

	bts := filehandler.NewBytesStore()
	buf := bytes.NewBuffer(make([]byte, 100_000))
	post_tckr := time.NewTicker(time.Millisecond * 2)
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
		intc := 0

		var d_c <-chan struct{}
	engine_loop:
		for _, file_ := range FileTree.FileState.Keys() {
			state, _ := FileTree.FileState.Get(file_)
			switch state {
			case dir_handler.FileStateToHash:
				d_c = HashingFunction(ctx.Add(), bts, file_)
			case dir_handler.FileStateToUpload:
				d_c = UploadingFunction(ctx.Add(), client, buf, file_)

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
			<-post_tckr.C
			intc += 1
			if intc > 3 {
				os.Exit(1)
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

func HashingFunction(ctx ftp_context.Context, bs *filehandler.BytesStore, file_p string) (d_c <-chan struct{}) {
	loc := logging.Loc("HashingFunction(bs *filehandler.BytesStore, file_p string) ")
	var err1, err2, err3 error
	fh, _ := FileTree.FileMap.Get(file_p)
	Logger.Logf(loc, "to hash: %s\nModTime: %s", file_p, fh.ModTime)
	bs.Reset()

	tmp_c := make(chan struct{}, 1)
	d_c = tmp_c

	go func() {
		defer ctx.Finished()
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
	}()
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
			Err:      true,
			After:    `received, ok := ts["received"]`,
			Message:  `didn't receive a hash from server`,
		})
		return
	}

	if fh.Hash != received {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location: string(loc),
			Err:      true,
			After:    `fh.Hash != received`,
			Message:  fmt.Sprintf(`received the wrong hash from server:\nfile: %s\nsent: %s\n received: %s`, fh.Path, fh.Hash, received),
		})
		return
	}

	route = ClientConfig.ServerIps[0] + "/upload/stream/" + fh.Hash

	data, err4 = os.ReadFile(file_p)
	if err4 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     fmt.Sprintf(`data, err4 = os.ReadFile(file_p:"%s")`, fh.Path),
			Message:   err4.Error(),
			CallStack: []error{err4},
		})
		return
	}

	tmp_2 := &map[string]any{
		"hash": fh.Hash,
		"data": base64.StdEncoding.EncodeToString(data),
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
	buf.Write(data)
	log.Println(string(data))

	res, err7 := client.Post(route, "application/json", buf)
	if err7 != nil {
		log.Fatalln(err7)
	}

	buf.Reset()
	n, err8 := buf.ReadFrom(res.Body)
	if err8 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     `n, err8 := buf.ReadFrom(res.Body)`,
			Message:   err8.Error(),
			CallStack: []error{err8},
		})
		return
	}
	res.Body.Close()

	data = buf.Bytes()[:n]

	err9 := json.Unmarshal(data, &tmp_2)
	if err9 != nil {
		Logger.LogErr(loc, &ftp_context.LogItem{
			Location:  string(loc),
			After:     `err9 := json.Unmarshal(buf.Bytes(),&tmp_2)`,
			Message:   err9.Error(),
			CallStack: []error{err9},
		})
		return
	}

	log.Println(string(data))

	Logger.Logf(loc, "done uploading: %s\nat: %s\response:\t%s", file_p, fmt.Sprint(time.Now()), string(resp))

	return
}

func DownloadingFunction(client *http.Client, file_p string) (d_c <-chan struct{}) {
	tmp_c := make(chan struct{}, 1)
	d_c = tmp_c
	defer close(tmp_c)

	return
}
