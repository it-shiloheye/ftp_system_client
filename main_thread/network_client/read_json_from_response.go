package netclient

import (
	"encoding/json"

	"io"

	"github.com/it-shiloheye/ftp_system/client/main_thread/logging"
	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
)

func read_json_from_response(r io.ReadCloser, tmp any) (out []byte, err ftp_context.LogErr) {
	loc := logging.Loc("read_json_from_buffer(r io.Reader,tmp any)(out []byte, err ftp_context.LogErr)")
	out, eror := io.ReadAll(r)
	if eror != nil {
		Logger.LogErr(loc, eror)
		err = ftp_context.NewLogItem(string(loc), true).
			SetAfter("out, eror = BufferStore.Read(res.Body)").
			SetMessage(eror.Error()).
			AppendParentError(eror)
		return
	}

	eror_ := json.Unmarshal(out, tmp)
	if eror_ != nil {
		Logger.LogErr(loc, eror_)
		err = ftp_context.NewLogItem(string(loc), true).
			SetAfter("json.Unmarshal(out, tmp)").
			SetMessage(eror.Error()).
			AppendParentError(eror)
		return
	}

	return
}
