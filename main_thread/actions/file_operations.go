package actions

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"

	"github.com/it-shiloheye/ftp_system_lib/base"
	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
)

func ReadJson(path string, val any) (err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	err = json.Unmarshal(b, val)
	return
}

func WriteJson(dir_path string, name string, val any) (err ftp_context.LogErr) {
	_text, _err := json.MarshalIndent(val, "", "\t")
	if _err != nil {
		err = ftp_context.NewLogItem("WriteJson", true).Set("after", "json.MarshalIndent").AppendParentError(_err)
		return
	}
	f_mode := fs.FileMode(base.S_IRWXU | base.S_IRWXO)

	_err = os.MkdirAll(dir_path, f_mode)
	if _err != nil && !errors.Is(err, os.ErrExist) {
		err = ftp_context.NewLogItem("WriteJson", true).Set("after", "os.MkdirAll").AppendParentError(_err)
		return
	}

	_err = os.WriteFile(dir_path+"\\"+name+".json", _text, f_mode)
	if err != nil {
		_err = ftp_context.NewLogItem("WriteJson", true).Set("after", "os.WriteFile").AppendParentError(_err)
		return
	}

	return
}
