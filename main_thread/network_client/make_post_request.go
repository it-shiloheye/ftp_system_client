package netclient

import (
	"io"
	"net/http"

	"github.com/it-shiloheye/ftp_system_client/main_thread/logging"
	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
)

func make_post_request(client *http.Client, route string, contentType string, body io.Reader) (res *http.Response, out []byte, err ftp_context.LogErr) {
	loc := logging.Loc("make_get_request(client *http.Client, route string, tmp any) (res *http.Response, out []byte, err ftp_context.LogErr)")
	var eror error

	res, eror = client.Post(route, contentType, body)
	if eror != nil {
		Logger.LogErr(loc, eror)
		return res, nil, ftp_context.NewLogItem(string(loc), true).
			SetAfter("client.Get").
			AppendParentError(eror)

	}
	// log.Println(loc, "client.Get(route)", "done", res)
	out, eror = io.ReadAll(res.Body)
	if eror != nil {
		Logger.LogErr(loc, eror)
		return res, nil, ftp_context.NewLogItem(string(loc), true).
			SetAfter("out, eror = io.ReadAll(res.Body)").
			SetMessage(eror.Error()).
			AppendParentError(eror)
	}

	return
}
