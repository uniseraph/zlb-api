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

func getDomainJson(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	name := mux.Vars(r)["name"]
	key := fmt.Sprintf(KEY_FORMAT,name);
	pair, _, err := client.KV().Get(key,nil )
	result := &ApiResult{}
        if err != nil  {
            result.State=STATE_FAIL
            result.Msg = fmt.Sprintf("getDomain Error :%s", err.Error())
 	} else if pair == nil {
            result.State=STATE_FAIL
            result.Msg= fmt.Sprintf("Can't find The Domain :%s", name)
        } else {
            var f interface{}
            err = json.Unmarshal(pair.Value, &f)            
            result.State=STATE_OK
            result.Data=f 
        }
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func getDomainList(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
	keys, _, err :=  client.KV().Keys("zlb_healthcheck/", "/", nil)
	var dynaArr []string
	if err == nil {
	     for _, key := range keys {
		v := strings.Split(key, "/")
		domainName := v[1]
		dynaArr = append(dynaArr, domainName)
	     }
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
	result := &ApiResult{}
        domainName := mux.Vars(r)["name"]
        if domainName == "" {
           result.State=STATE_FAIL
           result.Msg="Please set DomainName in URI "
        } else {
	_, err:= client.KV().Delete(fmt.Sprintf(KEY_FORMAT,domainName),nil)
        if err != nil {
		logrus.WithFields(logrus.Fields{
			"domainName": domainName,
		}).Infof("delete consule ", err.Error())
                result.State=STATE_FAIL
                result.Msg=fmt.Sprintf("delete fail :%s", err.Error())
	} else {
             result.State=STATE_OK
        }
        }
        w.WriteHeader(http.StatusOK)
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(result)
}

func putHealthCheckCfg(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	client, _ := ctx.Value(KEY_CONSUL_CLIENT).(*api.Client)
 	domainName := mux.Vars(r)["name"]
        result := &ApiResult{}
        if domainName != "" {	
        
        parasMap := make(map[string]string) 	
	var keys []string
	keys = append(keys,"type")
	keys = append(keys,"uri")
	keys = append(keys,"valid_statuses")
	keys = append(keys,"interval")
	keys = append(keys,"timeout")
	keys = append(keys,"fall")
	keys = append(keys,"rise")
	keys = append(keys,"concurrency")

	for i := 0; i <len(keys); i++ {
      	     v := r.PostFormValue(keys[i])
             if v != "" {
               parasMap[keys[i]] = r.PostFormValue(keys[i])
             }
	}

	jsonString,_ := json.Marshal(parasMap)
	_, err := client.KV().Put(&api.KVPair{
		Key:   fmt.Sprintf(KEY_FORMAT, domainName),
		Value: jsonString,
	}, nil)


       if err != nil {
		logrus.WithFields(logrus.Fields{"domainname": domainName,
			"jsonString": jsonString,}).Infof("put consule fail :%s", err.Error())
		result.State = STATE_FAIL
		result.Msg =  fmt.Sprintf("put consule fail :%s", err.Error())
	} else {
		result.State  = STATE_OK;
	}
        } else {
              result.State = STATE_FAIL
              result.Msg =  "Please set DomainName in URI"
        } 
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

var routers = map[string]map[string]Handler{
	"HEAD": {},
	"GET": {
	    "/zlb/domains":              getDomainList,
	    "/zlb/domains/{name:.*}": getDomainJson,
	},
	"POST": {},
	"PUT":     {
	    "/zlb/domains/{name:.*}": putHealthCheckCfg,
	},
	"DELETE":  {
	    "/zlb/domains/{name:.*}": deleteHealthCheckCfg,
	},
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
