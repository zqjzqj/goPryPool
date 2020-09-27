#暂只支持芝麻代理
    使用前添加IP白名单
    
    pool := OpenPool(&ZmProxy{})
    pool.SetMaxOpen(2)
    pry, err := pool.GetPry()
    if err != nil {
        log.Fatal(err)
    }

    log.Println(pry.GetProxyUrl())
    
#自定义代理
    实现接口
    type Driver interface {
    	CreateProxies(num int, pool *Pool) ([]*Proxy, error)
    }
