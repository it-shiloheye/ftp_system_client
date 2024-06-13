package main

import (
	"log"

	initialiseclient "github.com/it-shiloheye/ftp_system_client/init_client"
	mainthread "github.com/it-shiloheye/ftp_system_client/main_thread"
	"github.com/it-shiloheye/ftp_system_client/main_thread/dir_handler"
	"github.com/it-shiloheye/ftp_system_client/main_thread/logging"

	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
)

var ClientConfig = initialiseclient.ClientConfig

var Logger = logging.Logger

func init() {
	if len(ClientConfig.DataDir) < 1 {
		ClientConfig.DataDir = "./data"
	}
}

func main() {
	initialiseclient.InitialiseClientConfig()
	loc := logging.Loc("main")
	if ClientConfig == nil {
		log.Fatalln("no client config")
	}
	dir_handler.InitialiseFileTree(ClientConfig.DataDir + "/file-tree.json")
	logging.InitialiseLogging(ClientConfig.DataDir)

	Logger.Logf(loc, "new client started: %s", ClientConfig.ClientId)
	ctx := ftp_context.CreateNewContext()
	defer ctx.Wait()

	go Logger.Engine(ctx.Add(), ClientConfig.DataDir)
	go initialiseclient.UpdateClientConfig(ClientConfig.DataDir, ctx.Add())
	mainthread.MainThread(ctx.Add())

}
