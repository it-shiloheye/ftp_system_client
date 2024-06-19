package netclient

import (
	"encoding/json"

	"net/http"
	"net/http/cookiejar"

	"os"

	initialiseclient "github.com/it-shiloheye/ftp_system_client/init_client"

	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"

	"github.com/it-shiloheye/ftp_system_lib/logging/log_item"
	ftp_tlshandler "github.com/it-shiloheye/ftp_system_lib/tls_handler/v2"
)

var ClientConfig = initialiseclient.ClientConfig

type Route struct {
	BaseUrl  string `json:"base_url"`
	Pathname string `json:"path_name"`
}

func (r *Route) Url() string {
	return r.BaseUrl + r.Pathname
}

func NewNetworkClient(ctx ftp_context.Context) (cl *http.Client, err log_item.LogErr) {
	loc := log_item.Loc("NewNetworkClient(ctx ftp_context.Context)(cl *http.Client, err log_item.LogErr )")
	jar, err1 := cookiejar.New(&cookiejar.Options{})
	if err1 != nil {
		err = log_item.NewLogItem(loc, log_item.LogLevelError02).SetAfterf("jar, err1 := cookiejar.New(&cookiejar.Options{})").SetMessage(err1.Error()).AppendParentError(err1)
		return
	}
	cl = &http.Client{
		Jar: jar,
	}

	tmp, err1 := os.ReadFile("./data/certs/ca_certs.json")
	if err1 != nil {
		err = log_item.NewLogItem(loc, log_item.LogLevelError02).SetAfterf("tmp, err1 := os.ReadFile(%s)", "./certs/ca_certs.json").SetMessage(err1.Error()).AppendParentError(err1)
		return
	}

	ca := ftp_tlshandler.CAPem{}
	err2 := json.Unmarshal(tmp, &ca)
	if err2 != nil {
		err = log_item.NewLogItem(loc, log_item.LogLevelError02).SetAfterf("err2 := json.Unmarshal(tmp,&ca)").SetMessage(err2.Error()).AppendParentError(err2)
		return
	}

	client_tls_config := ftp_tlshandler.ClientTLSConf(ca)
	cl.Transport = &http.Transport{
		TLSClientConfig:   client_tls_config,
		MaxConnsPerHost:   20,
		DisableKeepAlives: false,
		ForceAttemptHTTP2: true,
	}

	return

}
