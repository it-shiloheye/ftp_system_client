package netclient

import (
	"encoding/json"
	"fmt"

	"net/http"
	"net/url"
	"os"

	initialiseclient "github.com/it-shiloheye/ftp_system_client/init_client"
	ftp_base "github.com/it-shiloheye/ftp_system_lib/base"
	ftp_context "github.com/it-shiloheye/ftp_system_lib/context"
	"github.com/it-shiloheye/ftp_system_lib/logging"
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

type CookieJar_ struct {
	cookies ftp_base.MutexedMap[[]*http.Cookie]
}

func NewCookieJar() *CookieJar_ {
	return &CookieJar_{
		cookies: ftp_base.NewMutexedMap[[]*http.Cookie](),
	}
}

func (cj *CookieJar_) Cookies(u *url.URL) []*http.Cookie {
	tmp := fmt.Sprint(u)

	cookies, ok := cj.cookies.Get(tmp)
	if !ok {
		return nil
	}
	return cookies
}
func (cj *CookieJar_) SetCookies(url_ *url.URL, cookies []*http.Cookie) {
	tmp := fmt.Sprint(url_)
	uniq_c := map[string]bool{}

	c_array, ok := cj.cookies.Get(tmp)
	if !ok {
		c_array1 := []*http.Cookie{}
		for _, c := range cookies {
			if _, uniq := uniq_c[fmt.Sprint(uniq_c)]; !uniq {
				c_array1 = append(c_array1, c)
			}

		}
		cj.cookies.Set(tmp, c_array1)
		return

	}
	c_array2 := []*http.Cookie{}
	for _, c := range append(cookies, c_array...) {
		if _, uniq := uniq_c[fmt.Sprint(uniq_c)]; !uniq {
			c_array2 = append(c_array2, c)
		}
	}

	cj.cookies.Set(tmp, c_array2)
}

func NewNetworkClient(ctx ftp_context.Context) (cl *http.Client, err ftp_context.LogErr) {
	loc := logging.Loc("NewNetworkClient(ctx ftp_context.Context)(cl *http.Client, err ftp_context.LogErr )")
	cl = &http.Client{
		Jar: NewCookieJar(),
	}

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
		TLSClientConfig:   client_tls_config,
		MaxConnsPerHost:   20,
		DisableKeepAlives: false,
		ForceAttemptHTTP2: true,
	}

	return

}
