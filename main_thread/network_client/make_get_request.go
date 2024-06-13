package netclient

import (
	"encoding/json"

	"io"
	"net/http"

	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
	"github.com/it-shiloheye/ftp_system_lib/logging"
)

var Logger = logging.Logger

func make_get_request(client *http.Client, route string, tmp any) (res *http.Response, out []byte, err ftp_context.LogErr) {
	loc := logging.Loc("make_get_request(client *http.Client, route string, tmp any) (res *http.Response, out []byte, err ftp_context.LogErr)")
	var eror error

	res, eror = client.Get(route)
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
	// log.Println(loc, string(out))
	eror = json.Unmarshal(out, tmp)
	if eror != nil {
		Logger.LogErr(loc, eror)
		return res, out, ftp_context.NewLogItem(string(loc), true).
			SetAfter("json.Unmarshal(out, tmp)").
			AppendParentError(eror)

	}

	return
}
