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

const KEY_FORMAT = "zlb_healthcheck/%s"
const STATE_OK = "OK"
const STATE_FAIL = "FAIL"

type Handler func(c context.Context, w http.ResponseWriter, r *http.Request)

type ApiResult struct {
     State string `json:"state"`
     Msg string   `json:"msg,omitempty"`
     Data interface{} `json:"data,omitempty"`
}

type HealthCheckCfg struct {
     Type string  `json:"type"`
     Uri string   `json:"uri,omitempty"`
     Valid_statuses string `json:"valid_statuses,omitempty"`
     Interval int  `json:"interval,omitempty"`
     Timeout int  `json:"timeout,omitempty"`
     Fall int    `json:"fall,omitempty"`
     Rise int    `json:"rise,omitempty"`
     Concurrency int `json:"concurrency,omitempty"`
}

func getDomainJson(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	name := mux.Vars(r)["name"]
	key := fmt.Sprintf(KEY_FORMAT,name);
	pair, _, err := client.KV().Get(key,nil )
	result := &ApiResult{}
        if err != nil  {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
 	} 
        if pair == nil {
            http.Error(w, fmt.Sprintf("Can't find The Domain :%s", name), 404)
            return
        } 
        var f interface{}
            err = json.Unmarshal(pair.Value, &f)            
            result.State=STATE_OK
            result.Data=f 
        
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func getDomainList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	keys, _, err :=  client.KV().Keys("zlb_healthcheck/", "/", nil)
	var dynaArr []string
	if err != nil {
            http.Error(w,err.Error(),http.StatusInternalServerError)
            return
        }
	for _, key := range keys {
		v := strings.Split(key, "/")
		domainName := v[1]
		dynaArr = append(dynaArr, domainName)
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	result := &ApiResult{
		State:STATE_OK,
		Data:dynaArr,
	};
        json.NewEncoder(w).Encode(result)
}



func deleteHealthCheckCfg(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
        domainName := mux.Vars(r)["name"]
        if domainName == "" {
           http.Error(w,"Please set DomainName in URI ",404)
           return;
        } 
	_, err:= client.KV().Delete(fmt.Sprintf(KEY_FORMAT,domainName),nil)
        if err != nil {
		logrus.WithFields(logrus.Fields{
			"domainName": domainName,
		}).Infof("delete consule ", err.Error())
	   http.Error(w,"Please set DomainName in URI ",http.StatusInternalServerError)
           return;
        } 
        result := &ApiResult{}
        result.State=STATE_OK
        w.WriteHeader(http.StatusOK)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
}

func putHealthCheckCfg(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
 	domainName := mux.Vars(r)["name"]
        result := &ApiResult{}
        if domainName == "" {
           http.Error(w,"Please set DomainName in URI ",404)
           return;
        }
        req := &HealthCheckCfg{}
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
           http.Error(w,err.Error(),http.StatusInternalServerError)
           return;
        } 
        jsonstr,_ :=json.Marshal(req)
        _, err := client.KV().Put(&api.KVPair{
		  Key:   fmt.Sprintf(KEY_FORMAT, domainName),
		  Value: jsonstr,
        }, nil)

        if err != nil {
		 logrus.WithFields(logrus.Fields{"domainname": domainName}).Infof("put consule fail :%s", err.Error())
          http.Error(w,err.Error(),http.StatusInternalServerError)
          return;
        }
        result.State  = STATE_OK
        w.WriteHeader(http.StatusOK)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
}

var routers = map[string]map[string]Handler{
	"HEAD": {},
	"GET": {},
        "POST": {
            "/zlb/domains/list":              getDomainList,
            "/zlb/domains/{name:.*}/inspect": getDomainJson,
            "/zlb/domains/{name:.*}/update": putHealthCheckCfg,
            "/zlb/domains/{name:.*}/remove": deleteHealthCheckCfg, 
        },
	"PUT":{},
	"DELETE":{},
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
