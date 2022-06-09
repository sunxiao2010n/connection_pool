/**召回接口*/
service SayHello{
    HelloRsp SayHello(1: HelloReq req);
}

struct HelloReq{
    1: string req,
}

struct HelloRsp{
    1: string rsp,
}

struct TItem{
    1: string id,
    2: double score,
}