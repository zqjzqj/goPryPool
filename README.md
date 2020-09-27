#暂只支持芝麻代理
    使用前添加IP白名单
    
    pool := OpenPool(&ZmProxy{})
    pool.SetMaxOpen(2)
    
    //该方法在无空闲代理的时候回阻塞 
    //所以其余代理使用后需要及时释放或关闭
    pry, err := pool.GetPry()
    if err != nil {
        log.Fatal(err)
    }

    log.Println(pry.GetProxyUrl())
    
    //释放代理 放回代理池
    pry.Release()
    pry.Close() 关闭代理 不在放回连接池
    //更多方法请查看源代码
#自定义代理
    实现接口
    
    type Driver interface {
    	CreateProxies(num int, pool *Pool) ([]*Proxy, error)
    }
