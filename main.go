package main

import (
	"log"
	"time"

	initialiseclient "github.com/it-shiloheye/ftp_system_client/init_client"
	mainthread "github.com/it-shiloheye/ftp_system_client/main_thread"
	"github.com/it-shiloheye/ftp_system_client/main_thread/dir_handler"
	"github.com/it-shiloheye/ftp_system_client/main_thread/logging"

	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
)

var ClientConfig = initialiseclient.ClientConfig

var Logger = logging.Logger

func main() {
	loc := logging.Loc("main")
	if ClientConfig == nil {
		log.Fatalln("no client config")
	}
	dir_handler.InitialiseFileTree(ClientConfig.DataDir + "/file-tree.json")

	Logger.Logf(loc, "new client started: %s", ClientConfig.Id)
	ctx := ftp_context.CreateNewContext()
	defer ctx.Wait()

	go Logger.Engine(ctx.Add())
	go UpdateConfig(ctx.Add())
	mainthread.MainThread(ctx.Add())

}

func UpdateConfig(ctx ftp_context.Context) {
	loc := logging.Loc("UpdateConfig(ctx ftp_context.Context)")
	defer ctx.Finished()
	tc := time.NewTicker(time.Minute)
	for ok := true; ok; {
		select {
		case <-tc.C:
		case _, ok = <-ctx.Done():
		}

		err := initialiseclient.WriteConfig()
		if err != nil {
			Logger.LogErr(loc, err)
		}
		Logger.Logf(loc, "updated config successfully")
	}
}
