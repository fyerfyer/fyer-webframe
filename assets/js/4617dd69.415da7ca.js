"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[130],{8453:(e,r,n)=>{n.d(r,{R:()=>s,x:()=>d});var t=n(6540);const i={},l=t.createContext(i);function s(e){const r=t.useContext(l);return t.useMemo((function(){return"function"==typeof e?e(r):{...r,...e}}),[r,e])}function d(e){let r;return r=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:s(e.components),t.createElement(l.Provider,{value:r},e.children)}},9378:(e,r,n)=>{n.r(r),n.d(r,{assets:()=>a,contentTitle:()=>d,default:()=>h,frontMatter:()=>s,metadata:()=>t,toc:()=>c});const t=JSON.parse('{"id":"web/server/server","title":"Server","description":"\u670d\u52a1\u5668\u63d0\u4f9b\u4e86\u7075\u6d3b\u7684 HTTP \u670d\u52a1\u914d\u7f6e\u3001\u4f18\u96c5\u7684\u542f\u52a8\u548c\u5173\u95ed\u673a\u5236\u4ee5\u53ca\u4e30\u5bcc\u7684\u914d\u7f6e\u9009\u9879\u3002","source":"@site/docs/web/server/server.md","sourceDirName":"web/server","slug":"/web/server/","permalink":"/fyer-webframe/docs/web/server/","draft":false,"unlisted":false,"editUrl":"https://github.com/fyerfyer/fyer-webframe/tree/main/docs/web/server/server.md","tags":[],"version":"current","frontMatter":{},"sidebar":"tutorialSidebar","previous":{"title":"Router","permalink":"/fyer-webframe/docs/web/router/"}}');var i=n(4848),l=n(8453);const s={},d="Server",a={},c=[{value:"\u57fa\u672c\u914d\u7f6e",id:"\u57fa\u672c\u914d\u7f6e",level:2},{value:"\u521b\u5efa\u670d\u52a1\u5668",id:"\u521b\u5efa\u670d\u52a1\u5668",level:3},{value:"\u670d\u52a1\u5668\u63a5\u53e3",id:"\u670d\u52a1\u5668\u63a5\u53e3",level:3},{value:"\u6838\u5fc3\u7ec4\u4ef6",id:"\u6838\u5fc3\u7ec4\u4ef6",level:3},{value:"\u57fa\u7840\u4f7f\u7528\u793a\u4f8b",id:"\u57fa\u7840\u4f7f\u7528\u793a\u4f8b",level:3},{value:"\u4f18\u96c5\u5173\u95ed\u673a\u5236",id:"\u4f18\u96c5\u5173\u95ed\u673a\u5236",level:2},{value:"\u5b9e\u73b0\u539f\u7406",id:"\u5b9e\u73b0\u539f\u7406",level:3},{value:"\u4f7f\u7528\u65b9\u6cd5",id:"\u4f7f\u7528\u65b9\u6cd5",level:3},{value:"\u8d44\u6e90\u91ca\u653e\u6d41\u7a0b",id:"\u8d44\u6e90\u91ca\u653e\u6d41\u7a0b",level:3},{value:"\u9009\u9879\u6a21\u5f0f",id:"\u9009\u9879\u6a21\u5f0f",level:2},{value:"\u4ec0\u4e48\u662f\u9009\u9879\u6a21\u5f0f\uff1f",id:"\u4ec0\u4e48\u662f\u9009\u9879\u6a21\u5f0f",level:3},{value:"WebFrame \u4e2d\u7684\u9009\u9879\u6a21\u5f0f",id:"webframe-\u4e2d\u7684\u9009\u9879\u6a21\u5f0f",level:3},{value:"\u53ef\u7528\u9009\u9879",id:"\u53ef\u7528\u9009\u9879",level:3},{value:"1. <code>WithReadTimeout</code> - \u8bbe\u7f6e\u8bfb\u53d6\u8d85\u65f6",id:"1-withreadtimeout---\u8bbe\u7f6e\u8bfb\u53d6\u8d85\u65f6",level:4},{value:"2. <code>WithWriteTimeout</code> - \u8bbe\u7f6e\u5199\u5165\u8d85\u65f6",id:"2-withwritetimeout---\u8bbe\u7f6e\u5199\u5165\u8d85\u65f6",level:4},{value:"3. <code>WithTemplate</code> - \u8bbe\u7f6e\u6a21\u677f\u5f15\u64ce",id:"3-withtemplate---\u8bbe\u7f6e\u6a21\u677f\u5f15\u64ce",level:4},{value:"4. <code>WithNotFoundHandler</code> - \u81ea\u5b9a\u4e49 404 \u5904\u7406\u5668",id:"4-withnotfoundhandler---\u81ea\u5b9a\u4e49-404-\u5904\u7406\u5668",level:4},{value:"5. <code>WithBasePath</code> - \u8bbe\u7f6e\u57fa\u7840\u8def\u5f84\u524d\u7f00",id:"5-withbasepath---\u8bbe\u7f6e\u57fa\u7840\u8def\u5f84\u524d\u7f00",level:4},{value:"6. <code>WithPoolManager</code> - \u8bbe\u7f6e\u8fde\u63a5\u6c60\u7ba1\u7406\u5668",id:"6-withpoolmanager---\u8bbe\u7f6e\u8fde\u63a5\u6c60\u7ba1\u7406\u5668",level:4},{value:"\u94fe\u5f0f\u914d\u7f6e\u793a\u4f8b",id:"\u94fe\u5f0f\u914d\u7f6e\u793a\u4f8b",level:3},{value:"\u81ea\u5b9a\u4e49\u9009\u9879",id:"\u81ea\u5b9a\u4e49\u9009\u9879",level:3}];function o(e){const r={code:"code",h1:"h1",h2:"h2",h3:"h3",h4:"h4",header:"header",li:"li",ol:"ol",p:"p",pre:"pre",strong:"strong",...(0,l.R)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(r.header,{children:(0,i.jsx)(r.h1,{id:"server",children:"Server"})}),"\n",(0,i.jsx)(r.p,{children:"\u670d\u52a1\u5668\u63d0\u4f9b\u4e86\u7075\u6d3b\u7684 HTTP \u670d\u52a1\u914d\u7f6e\u3001\u4f18\u96c5\u7684\u542f\u52a8\u548c\u5173\u95ed\u673a\u5236\u4ee5\u53ca\u4e30\u5bcc\u7684\u914d\u7f6e\u9009\u9879\u3002"}),"\n",(0,i.jsx)(r.h2,{id:"\u57fa\u672c\u914d\u7f6e",children:"\u57fa\u672c\u914d\u7f6e"}),"\n",(0,i.jsxs)(r.p,{children:["WebFrame \u670d\u52a1\u5668\u4ee5 ",(0,i.jsx)(r.code,{children:"HTTPServer"})," \u4e3a\u6838\u5fc3\uff0c\u5b9e\u73b0\u4e86 ",(0,i.jsx)(r.code,{children:"Server"})," \u63a5\u53e3\uff0c\u63d0\u4f9b\u4e86\u5b8c\u6574\u7684 HTTP \u670d\u52a1\u80fd\u529b\u3002"]}),"\n",(0,i.jsx)(r.h3,{id:"\u521b\u5efa\u670d\u52a1\u5668",children:"\u521b\u5efa\u670d\u52a1\u5668"}),"\n",(0,i.jsx)(r.p,{children:"\u521b\u5efa\u4e00\u4e2a\u57fa\u672c\u7684 WebFrame \u670d\u52a1\u5668\u975e\u5e38\u7b80\u5355\uff1a"}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:'import "github.com/fyerfyer/fyer-webframe/web"\r\n\r\nfunc main() {\r\n    // \u521b\u5efa\u4e00\u4e2a\u65b0\u7684 HTTP \u670d\u52a1\u5668\r\n    server := web.NewHTTPServer()\r\n    \r\n    // \u6ce8\u518c\u8def\u7531\r\n    server.Get("/", func(ctx *web.Context) {\r\n        ctx.String(200, "Hello, WebFrame!")\r\n    })\r\n    \r\n    // \u542f\u52a8\u670d\u52a1\u5668\r\n    server.Start(":8080")\r\n}\n'})}),"\n",(0,i.jsx)(r.h3,{id:"\u670d\u52a1\u5668\u63a5\u53e3",children:"\u670d\u52a1\u5668\u63a5\u53e3"}),"\n",(0,i.jsxs)(r.p,{children:[(0,i.jsx)(r.code,{children:"Server"})," \u63a5\u53e3\u5b9a\u4e49\u4e86 WebFrame \u670d\u52a1\u5668\u5e94\u5177\u5907\u7684\u6838\u5fc3\u529f\u80fd\uff1a"]}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:"type Server interface {\r\n    http.Handler\r\n    Start(addr string) error\r\n    Shutdown(ctx context.Context) error\r\n    \r\n    // \u8def\u7531\u6ce8\u518c\u65b9\u6cd5\r\n    Get(path string, handler HandlerFunc) RouteRegister\r\n    Post(path string, handler HandlerFunc) RouteRegister\r\n    Put(path string, handler HandlerFunc) RouteRegister\r\n    Delete(path string, handler HandlerFunc) RouteRegister\r\n    Patch(path string, handler HandlerFunc) RouteRegister\r\n    Options(path string, handler HandlerFunc) RouteRegister\r\n    \r\n    // \u8def\u7531\u7ec4\u548c\u4e2d\u95f4\u4ef6\r\n    Group(prefix string) RouteGroup\r\n    Middleware() MiddlewareManager\r\n    \r\n    // \u6a21\u677f\u5f15\u64ce\r\n    UseTemplate(tpl Template) Server\r\n    GetTemplateEngine() Template\r\n}\n"})}),"\n",(0,i.jsx)(r.h3,{id:"\u6838\u5fc3\u7ec4\u4ef6",children:"\u6838\u5fc3\u7ec4\u4ef6"}),"\n",(0,i.jsx)(r.p,{children:"\u670d\u52a1\u5668\u7531\u4ee5\u4e0b\u6838\u5fc3\u7ec4\u4ef6\u7ec4\u6210\uff1a"}),"\n",(0,i.jsxs)(r.ol,{children:["\n",(0,i.jsxs)(r.li,{children:[(0,i.jsx)(r.strong,{children:"\u8def\u7531\u7cfb\u7edf"}),": \u5904\u7406 HTTP \u8bf7\u6c42\u8def\u7531\u548c\u5206\u53d1"]}),"\n",(0,i.jsxs)(r.li,{children:[(0,i.jsx)(r.strong,{children:"\u4e2d\u95f4\u4ef6\u94fe"}),": \u5904\u7406\u8bf7\u6c42\u524d\u540e\u7684\u903b\u8f91"]}),"\n",(0,i.jsxs)(r.li,{children:[(0,i.jsx)(r.strong,{children:"\u4e0a\u4e0b\u6587\u7ba1\u7406"}),": \u5c01\u88c5\u8bf7\u6c42\u548c\u54cd\u5e94\u64cd\u4f5c"]}),"\n",(0,i.jsxs)(r.li,{children:[(0,i.jsx)(r.strong,{children:"\u6a21\u677f\u5f15\u64ce"}),": \u63d0\u4f9b\u9875\u9762\u6e32\u67d3\u80fd\u529b"]}),"\n",(0,i.jsxs)(r.li,{children:[(0,i.jsx)(r.strong,{children:"\u8fde\u63a5\u6c60\u7ba1\u7406"}),": \u7ba1\u7406\u6570\u636e\u5e93\u3001Redis \u7b49\u8fde\u63a5\u8d44\u6e90"]}),"\n"]}),"\n",(0,i.jsx)(r.h3,{id:"\u57fa\u7840\u4f7f\u7528\u793a\u4f8b",children:"\u57fa\u7840\u4f7f\u7528\u793a\u4f8b"}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:'package main\r\n\r\nimport (\r\n    "log"\r\n    "github.com/fyerfyer/fyer-webframe/web"\r\n    "github.com/fyerfyer/fyer-webframe/web/middleware/accesslog"\r\n    "github.com/fyerfyer/fyer-webframe/web/middleware/recovery"\r\n)\r\n\r\nfunc main() {\r\n    // \u521b\u5efa HTTP \u670d\u52a1\u5668\r\n    server := web.NewHTTPServer()\r\n    \r\n    // \u6dfb\u52a0\u5168\u5c40\u4e2d\u95f4\u4ef6\r\n    server.Use("*", "*", recovery.Recovery())\r\n    server.Use("*", "*", accesslog.NewMiddlewareBuilder().Build())\r\n    \r\n    // \u6ce8\u518c\u6839\u8def\u7531\r\n    server.Get("/", func(ctx *web.Context) {\r\n        ctx.String(200, "Welcome to WebFrame!")\r\n    })\r\n    \r\n    // \u6ce8\u518c API \u8def\u7531\u7ec4\r\n    api := server.Group("/api")\r\n    \r\n    // \u6dfb\u52a0\u7528\u6237\u76f8\u5173\u8def\u7531\r\n    api.Get("/users", listUsers)\r\n    api.Post("/users", createUser)\r\n    api.Get("/users/:id", getUserByID)\r\n    \r\n    // \u542f\u52a8\u670d\u52a1\u5668\r\n    log.Println("Server starting on :8080")\r\n    if err := server.Start(":8080"); err != nil {\r\n        log.Fatalf("Server failed to start: %v", err)\r\n    }\r\n}\r\n\r\nfunc listUsers(ctx *web.Context) {\r\n    // \u5904\u7406\u83b7\u53d6\u7528\u6237\u5217\u8868\r\n    ctx.JSON(200, []map[string]any{\r\n        {"id": 1, "name": "User 1"},\r\n        {"id": 2, "name": "User 2"},\r\n    })\r\n}\r\n\r\nfunc createUser(ctx *web.Context) {\r\n    // \u5904\u7406\u521b\u5efa\u7528\u6237\r\n    // ...\r\n}\r\n\r\nfunc getUserByID(ctx *web.Context) {\r\n    // \u83b7\u53d6\u8def\u5f84\u53c2\u6570\r\n    id := ctx.PathParam("id").Value\r\n    // ...\r\n}\n'})}),"\n",(0,i.jsx)(r.h2,{id:"\u4f18\u96c5\u5173\u95ed\u673a\u5236",children:"\u4f18\u96c5\u5173\u95ed\u673a\u5236"}),"\n",(0,i.jsx)(r.p,{children:"\u670d\u52a1\u5668\u63d0\u4f9b\u4e86\u4f18\u96c5\u5173\u95ed\u673a\u5236\uff0c\u786e\u4fdd\u670d\u52a1\u5668\u5173\u95ed\u65f6\u80fd\u591f\u6b63\u786e\u5904\u7406\u73b0\u6709\u8bf7\u6c42\uff0c\u91ca\u653e\u8d44\u6e90\uff0c\u9632\u6b62\u8fde\u63a5\u6cc4\u6f0f\u3002"}),"\n",(0,i.jsx)(r.h3,{id:"\u5b9e\u73b0\u539f\u7406",children:"\u5b9e\u73b0\u539f\u7406"}),"\n",(0,i.jsx)(r.p,{children:"\u670d\u52a1\u5668\u7684\u4f18\u96c5\u5173\u95ed\u57fa\u4e8e\u4ee5\u4e0b\u673a\u5236\uff1a"}),"\n",(0,i.jsxs)(r.ol,{children:["\n",(0,i.jsxs)(r.li,{children:["\u4f7f\u7528 ",(0,i.jsx)(r.code,{children:"context.Context"})," \u63a7\u5236\u5173\u95ed\u8d85\u65f6"]}),"\n",(0,i.jsx)(r.li,{children:"\u7b49\u5f85\u6b63\u5728\u8fdb\u884c\u7684\u8bf7\u6c42\u5904\u7406\u5b8c\u6210"}),"\n",(0,i.jsx)(r.li,{children:"\u5173\u95ed\u6240\u6709\u8fde\u63a5\u6c60\u8d44\u6e90"}),"\n",(0,i.jsx)(r.li,{children:"\u9000\u51fa\u670d\u52a1\u5668\u8fdb\u7a0b"}),"\n"]}),"\n",(0,i.jsx)(r.h3,{id:"\u4f7f\u7528\u65b9\u6cd5",children:"\u4f7f\u7528\u65b9\u6cd5"}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:'package main\r\n\r\nimport (\r\n    "context"\r\n    "log"\r\n    "net/http"\r\n    "os"\r\n    "os/signal"\r\n    "syscall"\r\n    "time"\r\n    \r\n    "github.com/fyerfyer/fyer-webframe/web"\r\n)\r\n\r\nfunc main() {\r\n    // \u521b\u5efa\u670d\u52a1\u5668\r\n    server := web.NewHTTPServer()\r\n    \r\n    // \u914d\u7f6e\u8def\u7531...\r\n    \r\n    // \u521b\u5efa\u901a\u9053\u76d1\u542c\u7cfb\u7edf\u4fe1\u53f7\r\n    quit := make(chan os.Signal, 1)\r\n    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)\r\n    \r\n    // \u5728\u540e\u53f0\u542f\u52a8\u670d\u52a1\u5668\r\n    go func() {\r\n        log.Println("Server starting on :8080")\r\n        if err := server.Start(":8080"); err != nil && err != http.ErrServerClosed {\r\n            log.Fatalf("Server failed to start: %v", err)\r\n        }\r\n    }()\r\n    \r\n    // \u7b49\u5f85\u9000\u51fa\u4fe1\u53f7\r\n    <-quit\r\n    log.Println("Shutting down server...")\r\n    \r\n    // \u521b\u5efa\u4e0a\u4e0b\u6587\uff0c\u8bbe\u7f6e\u8d85\u65f6\u65f6\u95f4\r\n    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)\r\n    defer cancel()\r\n    \r\n    // \u4f18\u96c5\u5173\u95ed\u670d\u52a1\u5668\r\n    if err := server.Shutdown(ctx); err != nil {\r\n        log.Fatalf("Server forced to shutdown: %v", err)\r\n    }\r\n    \r\n    log.Println("Server gracefully stopped")\r\n}\n'})}),"\n",(0,i.jsx)(r.h3,{id:"\u8d44\u6e90\u91ca\u653e\u6d41\u7a0b",children:"\u8d44\u6e90\u91ca\u653e\u6d41\u7a0b"}),"\n",(0,i.jsxs)(r.p,{children:["\u5f53\u8c03\u7528 ",(0,i.jsx)(r.code,{children:"Shutdown"})," \u65b9\u6cd5\u65f6\uff0c\u670d\u52a1\u5668\u4f1a\u6309\u4ee5\u4e0b\u987a\u5e8f\u91ca\u653e\u8d44\u6e90\uff1a"]}),"\n",(0,i.jsxs)(r.ol,{children:["\n",(0,i.jsx)(r.li,{children:"\u505c\u6b62\u63a5\u6536\u65b0\u7684\u8fde\u63a5\u8bf7\u6c42"}),"\n",(0,i.jsx)(r.li,{children:"\u7b49\u5f85\u6240\u6709\u6d3b\u8dc3\u7684\u8bf7\u6c42\u5904\u7406\u5b8c\u6210"}),"\n",(0,i.jsx)(r.li,{children:"\u5173\u95ed\u6240\u6709\u8fde\u63a5\u6c60\u8d44\u6e90"}),"\n",(0,i.jsx)(r.li,{children:"\u91ca\u653e\u670d\u52a1\u5668\u8d44\u6e90"}),"\n"]}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:"// Shutdown \u4f18\u96c5\u5173\u95ed\r\nfunc (s *HTTPServer) Shutdown(ctx context.Context) error {\r\n    s.start = false\r\n\r\n    // \u5173\u95ed\u8fde\u63a5\u6c60\u7ba1\u7406\u5668\r\n    if s.poolManager != nil {\r\n        if err := s.poolManager.Shutdown(ctx); err != nil {\r\n            return err\r\n        }\r\n    }\r\n\r\n    return s.server.Shutdown(ctx)\r\n}\n"})}),"\n",(0,i.jsx)(r.h2,{id:"\u9009\u9879\u6a21\u5f0f",children:"\u9009\u9879\u6a21\u5f0f"}),"\n",(0,i.jsx)(r.p,{children:"\u670d\u52a1\u5668\u91c7\u7528\u9009\u9879\u6a21\u5f0f\u8fdb\u884c\u914d\u7f6e\uff0c\u63d0\u4f9b\u4e86\u7075\u6d3b\u4e14\u6613\u4e8e\u6269\u5c55\u7684\u914d\u7f6e\u65b9\u6cd5\u3002"}),"\n",(0,i.jsx)(r.h3,{id:"\u4ec0\u4e48\u662f\u9009\u9879\u6a21\u5f0f",children:"\u4ec0\u4e48\u662f\u9009\u9879\u6a21\u5f0f\uff1f"}),"\n",(0,i.jsx)(r.p,{children:"\u9009\u9879\u6a21\u5f0f\u662f\u4e00\u79cd\u51fd\u6570\u5f0f\u7f16\u7a0b\u6a21\u5f0f\uff0c\u901a\u8fc7\u5b9a\u4e49\u4e00\u7cfb\u5217\u914d\u7f6e\u51fd\u6570\u6765\u8bbe\u7f6e\u5bf9\u8c61\u7684\u5c5e\u6027\uff0c\u907f\u514d\u4f7f\u7528\u5927\u91cf\u7684\u6784\u9020\u51fd\u6570\u6216\u590d\u6742\u7684\u6784\u5efa\u5668\u6a21\u5f0f\u3002"}),"\n",(0,i.jsx)(r.h3,{id:"webframe-\u4e2d\u7684\u9009\u9879\u6a21\u5f0f",children:"WebFrame \u4e2d\u7684\u9009\u9879\u6a21\u5f0f"}),"\n",(0,i.jsxs)(r.p,{children:["\u5728 WebFrame \u4e2d\uff0c\u670d\u52a1\u5668\u9009\u9879\u901a\u8fc7 ",(0,i.jsx)(r.code,{children:"ServerOption"})," \u51fd\u6570\u7c7b\u578b\u5b9a\u4e49\uff1a"]}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:"// ServerOption \u5b9a\u4e49\u670d\u52a1\u5668\u9009\u9879\r\ntype ServerOption func(*HTTPServer)\n"})}),"\n",(0,i.jsx)(r.h3,{id:"\u53ef\u7528\u9009\u9879",children:"\u53ef\u7528\u9009\u9879"}),"\n",(0,i.jsx)(r.p,{children:"\u670d\u52a1\u5668\u63d0\u4f9b\u4ee5\u4e0b\u5185\u7f6e\u9009\u9879\uff1a"}),"\n",(0,i.jsxs)(r.h4,{id:"1-withreadtimeout---\u8bbe\u7f6e\u8bfb\u53d6\u8d85\u65f6",children:["1. ",(0,i.jsx)(r.code,{children:"WithReadTimeout"})," - \u8bbe\u7f6e\u8bfb\u53d6\u8d85\u65f6"]}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:"// \u8bbe\u7f6e 10 \u79d2\u8bfb\u53d6\u8d85\u65f6\r\nserver := web.NewHTTPServer(web.WithReadTimeout(10 * time.Second))\n"})}),"\n",(0,i.jsxs)(r.h4,{id:"2-withwritetimeout---\u8bbe\u7f6e\u5199\u5165\u8d85\u65f6",children:["2. ",(0,i.jsx)(r.code,{children:"WithWriteTimeout"})," - \u8bbe\u7f6e\u5199\u5165\u8d85\u65f6"]}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:"// \u8bbe\u7f6e 15 \u79d2\u5199\u5165\u8d85\u65f6\r\nserver := web.NewHTTPServer(web.WithWriteTimeout(15 * time.Second))\n"})}),"\n",(0,i.jsxs)(r.h4,{id:"3-withtemplate---\u8bbe\u7f6e\u6a21\u677f\u5f15\u64ce",children:["3. ",(0,i.jsx)(r.code,{children:"WithTemplate"})," - \u8bbe\u7f6e\u6a21\u677f\u5f15\u64ce"]}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:'// \u521b\u5efa\u6a21\u677f\u5f15\u64ce\r\ntpl := web.NewGoTemplate(web.WithPattern("./templates/*.html"))\r\n\r\n// \u8bbe\u7f6e\u5230\u670d\u52a1\u5668\r\nserver := web.NewHTTPServer(web.WithTemplate(tpl))\n'})}),"\n",(0,i.jsxs)(r.h4,{id:"4-withnotfoundhandler---\u81ea\u5b9a\u4e49-404-\u5904\u7406\u5668",children:["4. ",(0,i.jsx)(r.code,{children:"WithNotFoundHandler"})," - \u81ea\u5b9a\u4e49 404 \u5904\u7406\u5668"]}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:'// \u81ea\u5b9a\u4e49 404 \u5904\u7406\u5668\r\nnotFoundHandler := func(ctx *web.Context) {\r\n    ctx.HTML(404, "<h1>\u9875\u9762\u672a\u627e\u5230</h1><p>\u8bf7\u68c0\u67e5\u60a8\u7684 URL</p>")\r\n}\r\n\r\n// \u5e94\u7528\u81ea\u5b9a\u4e49\u5904\u7406\u5668\r\nserver := web.NewHTTPServer(web.WithNotFoundHandler(notFoundHandler))\n'})}),"\n",(0,i.jsxs)(r.h4,{id:"5-withbasepath---\u8bbe\u7f6e\u57fa\u7840\u8def\u5f84\u524d\u7f00",children:["5. ",(0,i.jsx)(r.code,{children:"WithBasePath"})," - \u8bbe\u7f6e\u57fa\u7840\u8def\u5f84\u524d\u7f00"]}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:'// \u6240\u6709\u8def\u7531\u90fd\u5c06\u4ee5 "/api/v1" \u4e3a\u524d\u7f00\r\nserver := web.NewHTTPServer(web.WithBasePath("/api/v1"))\n'})}),"\n",(0,i.jsxs)(r.h4,{id:"6-withpoolmanager---\u8bbe\u7f6e\u8fde\u63a5\u6c60\u7ba1\u7406\u5668",children:["6. ",(0,i.jsx)(r.code,{children:"WithPoolManager"})," - \u8bbe\u7f6e\u8fde\u63a5\u6c60\u7ba1\u7406\u5668"]}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:"// \u521b\u5efa\u8fde\u63a5\u6c60\u7ba1\u7406\u5668\r\npoolManager := myapp.NewPoolManager()\r\n\r\n// \u914d\u7f6e\u5230\u670d\u52a1\u5668\r\nserver := web.NewHTTPServer(web.WithPoolManager(poolManager))\n"})}),"\n",(0,i.jsx)(r.h3,{id:"\u94fe\u5f0f\u914d\u7f6e\u793a\u4f8b",children:"\u94fe\u5f0f\u914d\u7f6e\u793a\u4f8b"}),"\n",(0,i.jsx)(r.p,{children:"\u9009\u9879\u53ef\u4ee5\u7ec4\u5408\u4f7f\u7528\uff0c\u5b9e\u73b0\u94fe\u5f0f\u914d\u7f6e\uff1a"}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:'server := web.NewHTTPServer(\r\n    web.WithReadTimeout(10 * time.Second),\r\n    web.WithWriteTimeout(15 * time.Second),\r\n    web.WithBasePath("/api/v1"),\r\n    web.WithTemplate(tpl),\r\n    web.WithNotFoundHandler(customNotFoundHandler),\r\n)\n'})}),"\n",(0,i.jsx)(r.h3,{id:"\u81ea\u5b9a\u4e49\u9009\u9879",children:"\u81ea\u5b9a\u4e49\u9009\u9879"}),"\n",(0,i.jsx)(r.p,{children:"\u60a8\u53ef\u4ee5\u6839\u636e\u9700\u8981\u521b\u5efa\u81ea\u5df1\u7684\u9009\u9879\u51fd\u6570\uff1a"}),"\n",(0,i.jsx)(r.pre,{children:(0,i.jsx)(r.code,{className:"language-go",children:"// WithCustomLogger \u81ea\u5b9a\u4e49\u65e5\u5fd7\u8bb0\u5f55\u5668\u9009\u9879\r\nfunc WithCustomLogger(logger *log.Logger) web.ServerOption {\r\n    return func(server *web.HTTPServer) {\r\n        // \u8bbe\u7f6e\u81ea\u5b9a\u4e49\u65e5\u5fd7\u8bb0\u5f55\u5668\r\n        // \u6ce8\u610f\uff1a\u9700\u8981\u5728 HTTPServer \u7ed3\u6784\u4e2d\u6dfb\u52a0\u76f8\u5e94\u7684\u5b57\u6bb5\r\n    }\r\n}\r\n\r\n// \u4f7f\u7528\u81ea\u5b9a\u4e49\u9009\u9879\r\nserver := web.NewHTTPServer(WithCustomLogger(myCustomLogger))\n"})})]})}function h(e={}){const{wrapper:r}={...(0,l.R)(),...e.components};return r?(0,i.jsx)(r,{...e,children:(0,i.jsx)(o,{...e})}):o(e)}}}]);