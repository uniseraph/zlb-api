##ZLB-API 相关接口说明

注意：所有API接口目前只支持HTTP POST 访问

* 后端服务健康检查接口API
    *  获取所有支持健康检查的域名列表(/zlb/domains/list)
```
请求：curl -X POST http://127.0.0.1:6300/zlb/domains/list
响应：{"state":"OK","data":["a.com","b.com"]}
```
    *  得到某个域名对应后端服务健康检查的相关配置信息(zlb/domains/${domainName}/inspect)
``` 
请求: curl -X POST http://127.0.0.1:6300/zlb/domains/a.com/inspect
响应: {"state":"OK","data":{"type":"http","uri":"/health","valid_statuses","200,302"}}
```
    *  更新某个域名对应的后端服务健康检查配置信息(zlb/domains/${domainName}/update)
``` 
请求: curl -X POST --data "{"type":"http","uri":"/health","valid_statuses":"200,302"}' http://127.0.0.1:6300/zlb/domains/a.com/update
响应: {"state":"OK"}
```
关于健康检查配置信息的说明
```
type : 检查类型（http|tcp）
uri ：检查类型为http时，检查的uri路径。
valid_statuses ： 检查类型为http时，标记为有效的http返回状态码。多个状态码用,号隔开
interval : 健康检查的间隔时间，单位毫秒，默认为2000
timeout：健康检查的网络超时时间，单位毫秒，默认为1000
fall ： 检查时连续失败多少次计为该后端节点不可用，默认为3
ris ： 对于不可用节点检查成功后连续多少次将该节点恢复为健康状态，默认为2
concurrency : 健康检查时的并发线程数
```
    * 移除某个域名的健康检查项，不再对此域名对应后端服务节点进行健康检查(zlb/domains/${domainName}/remove)
``` 
请求：curl -X POST http://127.0.0.1/a.com/remove
响应：{"state":"OK"}
```
* Cookie拦截功能接口API (zlb/domains/${domainName}/setCookieFilter)
```
该功能主要实现对特定Cookie特定值的拦截  
请求：curl -X POST --data '{"name":"x-gray-tag","value","tag1","lifecylce":0}' POST http://127.0.0.1/a.com/setCookieFilter
参数说明：
name : Cookie 键名称
value: Cookie 键键值
lifecycle:  为0表示不拦截，大于0表示拦截
```
* 删除相关域名接口API (zlb/domains/${domainName}/destroy)
``` 
请求：curl -X POST http://127.0.0.1:6300/zlb/domains/a.com/destroy
响应：{"state":"OK"}
```
