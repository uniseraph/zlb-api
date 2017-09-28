##ZLB-API 相关接口说明

注意：所有API接口目前只支持HTTP POST 访问

* 后端服务健康检查接口API
    *  获取所有支持健康检查的域名列表(/zlb/domain/list)
```
请求：curl -X POST http://127.0.0.1:6300/zlb/domain/list
响应：["a.com","b.com"]
```
    *  得到某个域名对应的相关信息(zlb/domain/${domainName}/inspect)
``` 
请求: curl -X POST http://127.0.0.1:6300/zlb/domain/www.test.com/inspect
响应: {
    "zlb": {
        "www.test.com": {
            "cfg": {
                "/": "{\"Healthcheck\":{\"Type\":\"http\",\"Uri\":\"/health\",\"Valid_statuses\":\"404,200,302\"},\"KeepAlive\":1024}"
            },
            "server": {
                "/": {
                    "127.0.0.1:1031": ""
                },
                "/user": {
                    "127.0.0.1:1032": ""
                }
            }
        }
    }
}
```
    *  更新某个域名对应的后端服务健康检查配置信息(zlb/domains/${domainName}/update)
``` 
请求: curl --data '{"Healthcheck":{"Type":"http","Uri":"/health","Valid_statuses":"404,200,302"},"KeepAlive":1024,"Sticky":false}' http://127.0.0.1:6300/zlb/domains/a.com/update
响应: ok
```
关于健康检查配置信息的说明
```
Type : 检查类型（http|tcp）
Uri ：检查类型为http时，检查的uri路径。
Valid_statuses ： 检查类型为http时，标记为有效的http返回状态码。多个状态码用,号隔开
Interval : 健康检查的间隔时间，单位毫秒，默认为2000
Timeout：健康检查的网络超时时间，单位毫秒，默认为1000
Fall ： 检查时连续失败多少次计为该后端节点不可用，默认为3
Ris ： 对于不可用节点检查成功后连续多少次将该节点恢复为健康状态，默认为2
Concurrency : 健康检查时的并发线程数
KeepAlive : 与后端服务保持长连接的个数,可选，默认为10
Sticky: 是否需要session粘滞

```
    *  新建某个域名对应的相关配置信息(zlb/domains/${domainName}/create) 
```
    该部分参数和返回值与update接口一致
```

    * 移除某个域名的健康检查项，不再对此域名对应后端服务节点进行健康检查(zlb/domains/${domainName}/remove)
``` 
请求：curl -X POST http://127.0.0.1:6300/zlb/domains/a.com/remove
响应：ok
```
* Cookie拦截功能接口API (zlb/cookie/${domainName}/setCookieFilter)
```
该功能主要实现对特定Cookie特定值的拦截  
请求：curl -X POST --data '{"name":"x-gray-tag","value","tag1","lifecylce":0}'  http://127.0.0.1:6300/zlb/domains/a.com/setCookieFilter
参数说明：
name : Cookie 键名称
value: Cookie 键键值
lifecycle:  为0表示不拦截，大于0表示拦截
```
