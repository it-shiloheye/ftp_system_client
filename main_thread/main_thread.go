package mainthread

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	initialiseclient "github.com/it-shiloheye/ftp_system_client/init_client"

	dir_handler "github.com/it-shiloheye/ftp_system_client/main_thread/dir_handler"
	netclient "github.com/it-shiloheye/ftp_system_client/main_thread/network_client"
	"github.com/it-shiloheye/ftp_system_lib/logging"
	"github.com/it-shiloheye/ftp_system_lib/logging/log_item"

	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
)

var ClientConfig = initialiseclient.ClientConfig
var Logger = logging.Logger
var FileTree = dir_handler.FileTree

func ticker(loc log_item.Loc, i int) {

	Logger.Logf(loc, "loc:\t%d", i)
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

		ticker(loc, 2)
		dir_data := ClientConfig.DirConfig
		exclude_list := dir_handler.GetExcludeList(dir_data.ExcludeDirs, dir_data.ExcludeRegex, dir_data.ExcluedFile)
		err1 := dir_handler.DirHandler(child_ctx.Add(), dir_data.Path, exclude_list)
		ticker(loc, 3)
		if err1 != nil {
			err := Logger.LogErr(loc, err1)
			log.Fatalln(err)
			continue
		}
		dir_handler.WriteFileTree(ctx.Add(), ClientConfig.DataDir)
		ticker(loc, 4)
		_, err_01 := ConfirmFunction(net_engine, FileTree)
		if err_01 != nil {
			err := Logger.LogErr(loc, err_01)
			log.Fatalln(err)
			continue
		}
		ticker(loc, 5)
		err_02 := UploadFunction(ctx.Add(), net_engine, FileTree)
		if err_02 != nil {
			err := Logger.LogErr(loc, err_02)
			log.Fatalln(err)
			continue
		}
		ticker(loc, 6)
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

func DownloadingFunction(client *http.Client, file_p string) error {

	return nil
}
