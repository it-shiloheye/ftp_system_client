package mainthread

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"os"

	// "strings"

	"log"
	"time"

	"github.com/google/uuid"
	initialiseclient "github.com/it-shiloheye/ftp_system_client/init_client"

	// "github.com/it-shiloheye/ftp_system_client/main_thread/actions"
	dir_handler "github.com/it-shiloheye/ftp_system_client/main_thread/dir_handler"
	netclient "github.com/it-shiloheye/ftp_system_client/main_thread/network_client"
	"github.com/it-shiloheye/ftp_system_lib/logging"
	"github.com/it-shiloheye/ftp_system_lib/logging/log_item"

	ftp_base "github.com/it-shiloheye/ftp_system_lib/base"
	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
	filehandler "github.com/it-shiloheye/ftp_system_lib/file_handler/v2"
)

var ClientConfig = initialiseclient.ClientConfig
var Logger = logging.Logger
var FileTree = dir_handler.FileTree

func ticker(loc log_item.Loc, i int) {

	// Logger.Logf(loc, "%d", i)
}

func MainThread(ctx ftp_context.Context) context.Context {
	loc := log_item.Loc("MainThread(ctx ftp_context.Context) context.Context ")

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

	if len(ClientConfig.ClientId) < 1 {
		ClientConfig.ClientId = uuid.NewString()
	}

	net_engine, err1 := test_server_connection(client, base_server)
	if err1 != nil {
		log.Fatalln(err1)
	}
	net_engine.SetCookie("/", &http.Cookie{
		Name:   "client-id",
		MaxAge: 0,
		Value:  ClientConfig.ClientId,
	})

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

		for _, file_ := range FileTree.FileState.Keys() {
			state, _ := FileTree.FileState.Get(file_)
			switch state {
			case dir_handler.FileStateToHash:

				HashingFunction(ctx.Add(), bts, file_)
			case dir_handler.FileStateToUpload:
				UploadingFunction(ctx.Add(), client, buf, file_)

			case dir_handler.FileStateToDownload:
				DownloadingFunction(client, file_)
			}

			<-post_tckr.C
			fh, ok := FileTree.FileMap.Get(file_)
			if ok {

				ConfirmFunction(client, fh.Hash)
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

func test_server_connection(client *http.Client, host string) (ne *netclient.NetworkEngine, err error) {
	loc := log_item.Locf(`test_server_connection(client *http.Client, host: "%s") (ne *netclient.NetworkEngine, err error)`, host)
	Logger.Logf(loc, "pinging server")
	ne = netclient.NewNetworkEngine(client, host)
	err1 := ne.Ping("/ping", 5)
	if err1 != nil {
		err = Logger.LogErr(loc, err1)
		return
	}

	Logger.Logf(loc, "server connected successfully: %s", host)
	return
}

func HashingFunction(ctx ftp_context.Context, bs *filehandler.BytesStore, file_p string) error {
	loc := log_item.Loc("HashingFunction(bs *filehandler.BytesStore, file_p string) ")
	var err1, err2, err3 error
	fh, _ := FileTree.FileMap.Get(file_p)
	Logger.Logf(loc, "to hash: %s\nModTime: %s", file_p, fh.ModTime)
	bs.Reset()

	defer ctx.Finished()
	if fh.File == nil {
		fh.File, err1 = ftp_base.OpenFile(file_p, os.O_RDONLY)
		if err1 != nil {

			return Logger.LogErr(loc, &log_item.LogItem{
				Location:  loc,
				After:     fmt.Sprintf(`fh.File, err1 = ftp_base.OpenFile("%s",os.O_RDONLY)`, file_p),
				Message:   err1.Error(),
				CallStack: []error{err1},
			})
		}
	}

	fh.Size, err2 = bs.ReadFrom(fh.File)
	if err2 != nil {

		return Logger.LogErr(loc, &log_item.LogItem{
			Location:  loc,
			After:     `_, err1 = bs.ReadFrom(fh.File)`,
			Message:   err2.Error(),
			CallStack: []error{err2},
		})
	}

	fh.Hash, err3 = bs.Hash()
	if err3 != nil {
		return Logger.LogErr(loc, &log_item.LogItem{
			Location:  loc,
			After:     `fh.Hash, err2 = bs.Hash()`,
			Message:   err3.Error(),
			CallStack: []error{err3},
		})

	}

	FileTree.FileState.Set(file_p, dir_handler.FileStateToUpload)
	Logger.Logf(loc, "done hashing: %s\nat: %s\nhash:\t%s", file_p, fmt.Sprint(time.Now()), fh.Hash)
	return nil
}

func UploadingFunction(ctx ftp_context.Context, client *http.Client, buf *bytes.Buffer, file_p string) error {
	loc := log_item.Loc(`UploadingFunction(client *http.Client, file_p string)`)

	client_id := ClientConfig.ClientId
	dir_id := ClientConfig.DirConfig.DirId

	fh, _ := FileTree.FileMap.Get(file_p)
	route := ClientConfig.ServerIps[0] + "/upload/file/" + fh.Hash
	if fh.MetaData == nil {
		fh.MetaData = make(map[string]any)
	}
	fh.Set("client-id", client_id)
	fh.Set("dir-id", dir_id)
	var err7 error
	data, err1 := json.Marshal(fh)
	if err1 != nil {

		return Logger.LogErr(loc, err1)
	}

	buf.Reset()

	_, err2 := buf.Write(data)
	if err2 != nil {

		return Logger.LogErr(loc, err2)
	}

	res, err3 := client.Post(route, "application/json", buf)
	if err3 != nil {

		return Logger.LogErr(loc, err3)
	}

	buf.Reset()
	buf.ReadFrom(res.Body)
	resp := buf.Bytes()

	ts := map[string]any{}

	err4 := json.Unmarshal(resp, &ts)
	if err4 != nil {

		return Logger.LogErr(loc, err4)
	}

	received, ok := ts["received"]
	if !ok {

		return Logger.LogErr(loc, &log_item.LogItem{
			Level:   log_item.LogLevelError02,
			After:   `received, ok := ts["received"]`,
			Message: `didn't receive a hash from server`,
		})
	}

	if fh.Hash != received {

		return Logger.LogErr(loc, &log_item.LogItem{
			After:   `fh.Hash != received`,
			Message: fmt.Sprintf(`received the wrong hash from server:\nfile: %s\nsent: %s\n received: %s`, fh.Path, fh.Hash, received),
		})
	}

	route = ClientConfig.ServerIps[0] + "/upload/stream/" + fh.Hash

	data, err4 = os.ReadFile(file_p)
	if err4 != nil {

		return Logger.LogErr(loc, err4)
	}

	tmp_2 := map[string]any{
		"hash": fh.Hash,
		"data": base64.StdEncoding.EncodeToString(data),
	}

	buf.Reset()
	err6 := json.NewEncoder(buf).Encode(&tmp_2)
	if err6 != nil {
		return Logger.LogErr(loc, err6)
	}

	for i := 0; ; i++ {
		res, err7 = client.Post(route, "application/json", buf)
		if err7 != nil {
			if errors.Is(err7, net.ErrClosed) {
				if i >= 5 {
					return Logger.LogErr(loc, err7)
				}
				<-time.After(time.Second)
				continue
			}
			return Logger.LogErr(loc, err7)
		}
		break
	}
	clear(tmp_2)

	err8 := json.NewDecoder(res.Body).Decode(&tmp_2)
	if err8 != nil {
		return Logger.LogErr(loc, err8)
	}
	res.Body.Close()

	if state, ok := tmp_2["state"]; ok {
		if s_state, ok := state.(string); ok && s_state == "success" {
			FileTree.FileState.Set(file_p, dir_handler.FileStateUploaded)
			Logger.Logf(loc, "done uploading: %s\nat: %s\response:\t%s", file_p, fmt.Sprint(time.Now()), string(resp))
			return nil
		}

	}

	return Logger.LogErr(loc, log_item.NewLogItem(loc, log_item.LogLevelError02).SetMessagef("failed uploading: %s\nat: %s\response:\t%s", file_p, fmt.Sprint(time.Now()), string(resp)))
}

func DownloadingFunction(client *http.Client, file_p string) error {

	return nil
}

func ConfirmFunction(client *http.Client, file_p string) error {
	loc := log_item.Locf(`ConfirmFunction(client *http.Client, file_hash:"%s") error`, file_p)

	fh, ok := FileTree.FileMap.Get(file_p)
	if !ok {
		FileTree.FileState.Set(file_p, dir_handler.FileStateToUpload)
		return nil
	}

	route := ClientConfig.ServerIps[0] + "/upload/confirm/" + fh.Hash
	// client_id := ClientConfig.ClientId
	// dir_id := ClientConfig.DirConfig.DirId

	res, err1 := client.Get(route)
	if err1 != nil {

		return Logger.LogErr(loc, &log_item.LogItem{
			Location:  loc,
			Time:      time.Now(),
			After:     fmt.Sprintf(`res, err1 := client.Get(route: %s)`, route),
			Message:   err1.Error(),
			CallStack: []error{err1},
		})
	}

	tmp := map[string]string{}

	d, err2 := io.ReadAll(res.Request.Body)
	res.Request.Body.Close()
	if err2 != nil {
		return Logger.LogErr(loc, &log_item.LogItem{
			Time:      time.Now(),
			Location:  loc,
			After:     `d, err2 := io.ReadAll(res.Request.Body)`,
			Message:   err2.Error(),
			CallStack: []error{err2},
		})
	}
	err3 := json.Unmarshal(d, &tmp)
	if err3 != nil {
		return Logger.LogErr(loc, &log_item.LogItem{
			Time:      time.Now(),
			Location:  loc,
			After:     `err3 := json.Unmarshal(d,&tmp)`,
			Message:   err3.Error(),
			CallStack: []error{err3},
		})
	}

	state, ok := tmp["state"]
	if !ok {
		return Logger.LogErr(loc, &log_item.LogItem{
			Location:  loc,
			Time:      time.Now(),
			After:     `state, ok := tmp["state"]`,
			Message:   fmt.Sprintf("server response is invalid:\n%s", string(d)),
			CallStack: []error{},
		})
	}
	switch state {
	case "missing":
		FileTree.FileState.Set(file_p, dir_handler.FileStateToUpload)
		return nil
	case "uploaded":
		return nil
	default:
		return Logger.LogErr(loc, &log_item.LogItem{
			Location:  loc,
			Time:      time.Now(),
			After:     `state, ok := tmp["state"]`,
			Message:   fmt.Sprintf("server response is invalid:\n%s", string(d)),
			CallStack: []error{},
		})
	}
}
