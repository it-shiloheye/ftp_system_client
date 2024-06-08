package netclient

import (
	"encoding/json"

	"net/http"
	"os"

	initialiseclient "github.com/it-shiloheye/ftp_system_client/init_client"
	"github.com/it-shiloheye/ftp_system_client/main_thread/logging"
	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
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

func MakeGetRequest(client *http.Client, route Route, tmp any) (out []byte, err ftp_context.LogErr) {
	loc := logging.Loc("MakeGetRequest(client *http.Client, route Route, tmp any) (out []byte, err ftp_context.LogErr)")
	Logger.Logf(loc, "make get request")
	_, out, err = make_get_request(client, route.Url(), tmp)
	return
}

func NewNetworkClient(ctx ftp_context.Context) (cl *http.Client, err ftp_context.LogErr) {
	loc := logging.Loc("NewNetworkClient(ctx ftp_context.Context)(cl *http.Client, err ftp_context.LogErr )")
	cl = &http.Client{}

	tmp, err1 := os.ReadFile("./data/certs/ca_certs.json")
	if err1 != nil {
		err = ftp_context.NewLogItem(string(loc), true).SetAfterf("tmp, err1 := os.ReadFile(%s)", "./certs/ca_certs.json").SetMessage(err1.Error()).AppendParentError(err1)
		return
	}

	ca := ftp_tlshandler.CAPem{}
	err2 := json.Unmarshal(tmp, &ca)
	if err2 != nil {
		err = ftp_context.NewLogItem(string(loc), true).SetAfterf("err2 := json.Unmarshal(tmp,&ca)").SetMessage(err2.Error()).AppendParentError(err2)
		return
	}

	client_tls_config := ftp_tlshandler.ClientTLSConf(ca)
	cl.Transport = &http.Transport{
		TLSClientConfig: client_tls_config,
	}

	return

}
