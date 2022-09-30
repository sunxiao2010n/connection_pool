/* rpc接口示例 */

service Helloworld{
    HelloRsp SayHello(1: HelloReq req);
}

struct HelloReq{
    1: string req,
}

struct HelloRsp{
    1: string rsp,
}
