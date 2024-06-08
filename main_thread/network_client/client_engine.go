package netclient

import (
	"net/http"

	dir_handler "github.com/it-shiloheye/ftp_system/client/main_thread/dir_handler"
	"github.com/it-shiloheye/ftp_system/client/main_thread/logging"
	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
)

var Logging = logging.Logger
var FileTree = dir_handler.FileTree

func NetClientEngine(ctx ftp_context.Context, client *http.Client) {
	defer ctx.Finished()

	for ok := true; ok; {
		select {
		case _, ok = <-ctx.Done():
		}
	}
}
