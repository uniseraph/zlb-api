package daemon

import (
	"context"
	"encoding/json"
	"encoding/base64"
	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/hashicorp/consul/api"
	"github.com/zanecloud/zlb/api/opts"
	"net/http"
	"strings"
)

const KEY_CONSUL_CLIENT = "consul.client"
const KEY_SERVER_OPTS = "server.opts"



type Handler func(c context.Context, w http.ResponseWriter, r *http.Request)

type HealthCheckCfg struct {
	Type           string `json:"Type"`
	Uri            string `json:"Uri,omitempty"`
	Valid_statuses string `json:"Valid_statuses,omitempty"`
	Interval       int    `json:"Interval,omitempty"`
	Timeout        int    `json:"Timeout,omitempty"`
	Fall           int    `json:"Fall,omitempty"`
	Rise           int    `json:"Rise,omitempty"`
	Concurrency    int    `json:"Concurrency,omitempty"`
}

type DomainCfg struct {
	Healthcheck  HealthCheckCfg `json:"Healthcheck"`
	Sticky bool  `json:"Sticky,omitempty"`
	KeepAlive int `json:"KeepAlive,omitempty"`
    Path string `json:"Path,omitempty"`
}

type CookieFilter struct {
	Name      string `json:"Name"`
	Value     string `json:"Value"`
	Lifecycle int64  `json:"Lifecycle"`
}

func explodeHelper(m map[string]interface{}, k, v, p string) error {
	if strings.Contains(k, "/") {
		parts := strings.Split(k, "/")
		fmt.Printf("%s= %d",k,len(parts))
		top := parts[0]
		if strings.HasPrefix(top,"path_") {
			udec,_ := base64.URLEncoding.DecodeString(top[5:])
		    top = string(udec)
		}
		key := strings.Join(parts[1:], "/")
		if _, ok := m[top]; !ok {
			m[top] = make(map[string]interface{})
		}
		nest, ok := m[top].(map[string]interface{})
		if !ok {
			return fmt.Errorf("not a map: %q: %q already has value %q", p, top, m[top])
		}
		return explodeHelper(nest, key, v, k)
	}

	if k != "" {
		if strings.HasPrefix(k,"path_"){
			udec,_:= base64.URLEncoding.DecodeString(k[5:]);
			k = string(udec)
		}
		m[k] = v
	}

	return nil
}

func getDomainJson(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	name := mux.Vars(r)["name"]
	pairs, _, err := client.KV().List("zlb/" + name + "/", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	m := make(map[string]interface{})
	for _, pair := range pairs {
		if err := explodeHelper(m, pair.Key, string(pair.Value) , pair.Key); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	jsonstr, _ := json.Marshal(m)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(jsonstr))
}

func getDomainList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	keys, _, err := client.KV().Keys("zlb/", "/", nil)
	var dynaArr []string
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, key := range keys {
		v := strings.Split(key, "/")
		domainName := v[1]
		dynaArr = append(dynaArr, domainName)
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	jsonstr, _ := json.Marshal(dynaArr)
	w.Write([]byte(jsonstr))
}



func updateDomain(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	domainName := mux.Vars(r)["name"]

	if domainName == "" {
		http.Error(w, "Please set DomainName in URI ", 404)
		return
	}
	req := &DomainCfg{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	path := req.Path
	if (path == "" ) {
	   	path = "/";
	}
	path = "path_" + base64.URLEncoding.EncodeToString([]byte(path))

	jsonstr, _ := json.Marshal(req)
	_, err := client.KV().Put(&api.KVPair{
		Key:   fmt.Sprintf("zlb/%s/cfg/%s", domainName,path),
		Value: jsonstr,
	}, nil)

	if err != nil {
		logrus.WithFields(logrus.Fields{"domainname": domainName}).Infof("put consule fail :%s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))

}

func setCookieFilter(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	domainName := mux.Vars(r)["name"]
	if domainName == "" {
		http.Error(w, "Please set DomainName in URI ", 404)
		return
	}
	req := &CookieFilter{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	consulkey := fmt.Sprintf("zlb/%s/ckfilter/%s/%s", domainName, req.Name, req.Value)
	_, err := client.KV().Put(&api.KVPair{
		Key:   consulkey,
		Value: []byte(fmt.Sprintf("%d", req.Lifecycle)),
	}, nil)

	if err != nil {
		logrus.WithFields(logrus.Fields{"consulkey": consulkey}).Infof("put consule fail :%s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func removeDomain(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	domainName := mux.Vars(r)["name"]
	if domainName == "" {
		http.Error(w, "Please set DomainName in URI ", 404)
		return
	}

	consulkey := fmt.Sprintf("zlb/%s", domainName)
	_, err := client.KV().DeleteTree(consulkey+"/", nil)

	if err != nil {
		logrus.WithFields(logrus.Fields{"consulkey": consulkey}).Infof("delete consule  fail :%s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("ok"))
}

var routers = map[string]map[string]Handler{
	"HEAD": {},
	"GET":  {},
	"POST": {
		"/zlb/domains/list":                      getDomainList,
		"/zlb/domains/{name:.*}/inspect":         getDomainJson,
		"/zlb/domains/{name:.*}/create":          updateDomain,
		"/zlb/domains/{name:.*}/update":          updateDomain,
		"/zlb/domains/{name:.*}/remove":          removeDomain,
		"/zlb/domains/{name:.*}/setCookieFilter": setCookieFilter,
	},
	"PUT":     {},
	"DELETE":  {},
	"OPTIONS": {},
}

func Run(opts opts.Options) {

	consulClient, err := api.NewClient(&api.Config{Address: opts.Consul})
	if err != nil {
		logrus.Fatalf("create a consul client error:%s", err.Error())
		return
	}

	r := mux.NewRouter()
	for method, mappings := range routers {
		for route, fct := range mappings {
			logrus.WithFields(logrus.Fields{"method": method, "route": route}).Debug("Registering HTTP route")

			localRoute := route
			localFct := fct
			wrap := func(w http.ResponseWriter, req *http.Request) {
				logrus.WithFields(logrus.Fields{"method": req.Method, "uri": req.RequestURI}).Debug("HTTP request received")

				ctx := context.WithValue(req.Context(), KEY_SERVER_OPTS, opts)
				ctx = context.WithValue(ctx, KEY_CONSUL_CLIENT, consulClient)

				localFct(ctx, w, req)
			}
			localMethod := method

			//r.Path("/v{version:[0-9.]+}" + localRoute).Methods(localMethod).HandlerFunc(wrap)
			r.Path(localRoute).Methods(localMethod).HandlerFunc(wrap)
		}
	}

	srv := http.Server{
		Handler: r,
		Addr:    opts.Address,
	}

	if err := srv.ListenAndServe(); err != nil {
		logrus.Errorf("run zlb api err:%s", err.Error())
	}
}
