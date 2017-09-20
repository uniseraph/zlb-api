package daemon

import (
	"context"
	"encoding/json"
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

const KEY_HEALTHCHECK_FORMAT = "zlb_healthcheck/%s"

//example zlb_healthcheck/a.com
const KEY_DOMAIN_FORMAT = "zlb_domain/%s"

//example zlb_domain/a.com
const KEY_COOKIEFILTER_FORMAT = "zlb_cookiefilter/%s/%s/%s"

//example zlb_cookiefilter/a.com/x-arg-tag/coupon
const STATE_OK = "OK"
const STATE_FAIL = "FAIL"

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

type CookieFilter struct {
	Name      string `json:"Name"`
	Value     string `json:"Value"`
	Lifecycle int64  `json:"Lifecycle"`
}

func getDomainJson(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	name := mux.Vars(r)["name"]
	key := fmt.Sprintf(KEY_HEALTHCHECK_FORMAT, name)
	pair, _, err := client.KV().Get(key, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if pair == nil {
		http.Error(w, fmt.Sprintf("Can't find The Domain :%s", name), 404)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(pair.Value)
}

func getDomainList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	keys, _, err := client.KV().Keys("zlb_healthcheck/", "/", nil)
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

func deleteHealthCheckCfg(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	domainName := mux.Vars(r)["name"]
	if domainName == "" {
		http.Error(w, "Please set DomainName in URI ", 404)
		return
	}
	_, err := client.KV().Delete(fmt.Sprintf(KEY_HEALTHCHECK_FORMAT, domainName), nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"domainName": domainName,
		}).Infof("delete consule ", err.Error())
		http.Error(w, "Please set DomainName in URI ", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func putHealthCheckCfg(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	domainName := mux.Vars(r)["name"]

	if domainName == "" {
		http.Error(w, "Please set DomainName in URI ", 404)
		return
	}
	req := &HealthCheckCfg{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonstr, _ := json.Marshal(req)
	_, err := client.KV().Put(&api.KVPair{
		Key:   fmt.Sprintf(KEY_HEALTHCHECK_FORMAT, domainName),
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

	consulkey := fmt.Sprintf(KEY_COOKIEFILTER_FORMAT, domainName, req.Name, req.Value)
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

func destroyDomain(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	domainName := mux.Vars(r)["name"]
	if domainName == "" {
		http.Error(w, "Please set DomainName in URI ", 404)
		return
	}

	consulkey := fmt.Sprintf(KEY_DOMAIN_FORMAT, domainName)
	_, err := client.KV().DeleteTree(consulkey+"/", nil)

	if err != nil {
		logrus.WithFields(logrus.Fields{"consulkey": consulkey}).Infof("delete consule  fail :%s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	consulkey = fmt.Sprintf(KEY_HEALTHCHECK_FORMAT, domainName)
	_, err = client.KV().Delete(consulkey, nil)
	if err != nil {
		logrus.WithFields(logrus.Fields{"consulkey": consulkey}).Infof("delete consule  fail :%s", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	consulkey = fmt.Sprintf("zlb_cookiefilter/%s/", domainName)
	_, err = client.KV().DeleteTree(consulkey, nil)
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
		"/zlb/healthcheck/list":                 getDomainList,
		"/zlb/healthcheck/{name:.*}/inspect":    getDomainJson,
		"/zlb/healthcheck/{name:.*}/update":     putHealthCheckCfg,
		"/zlb/healthcheck/{name:.*}/remove":     deleteHealthCheckCfg,
		"/zlb/cookie/{name:.*}/setCookieFilter": setCookieFilter,
		"/zlb/domain/{name:.*}/remove":          destroyDomain,
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
