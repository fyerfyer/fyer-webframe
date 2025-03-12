"use strict";(self.webpackChunkdocs=self.webpackChunkdocs||[]).push([[544],{7039:(r,e,n)=>{n.r(e),n.d(e,{assets:()=>d,contentTitle:()=>c,default:()=>o,frontMatter:()=>i,metadata:()=>t,toc:()=>l});const t=JSON.parse('{"id":"web/router/router","title":"Router","description":"WebFrame \u652f\u6301\u591a\u79cd\u8def\u7531\u6a21\u5f0f\u3001\u8def\u7531\u5206\u7ec4\u3001\u53c2\u6570\u63d0\u53d6\u548c\u9759\u6001\u8d44\u6e90\u670d\u52a1\u3002","source":"@site/docs/web/router/router.md","sourceDirName":"web/router","slug":"/web/router/","permalink":"/fyer-webframe/docs/web/router/","draft":false,"unlisted":false,"editUrl":"https://github.com/fyerfyer/fyer-webframe/tree/main/docs/web/router/router.md","tags":[],"version":"current","frontMatter":{},"sidebar":"tutorialSidebar","previous":{"title":"Request Handle","permalink":"/fyer-webframe/docs/web/request-handle/"},"next":{"title":"Server","permalink":"/fyer-webframe/docs/web/server/"}}');var s=n(4848),a=n(8453);const i={},c="Router",d={},l=[{value:"\u8def\u7531\u6ce8\u518c",id:"\u8def\u7531\u6ce8\u518c",level:2},{value:"\u57fa\u7840\u8def\u7531\u6ce8\u518c",id:"\u57fa\u7840\u8def\u7531\u6ce8\u518c",level:3},{value:"\u94fe\u5f0f API",id:"\u94fe\u5f0f-api",level:3},{value:"\u8def\u7531\u7ec4",id:"\u8def\u7531\u7ec4",level:2},{value:"\u521b\u5efa\u8def\u7531\u7ec4",id:"\u521b\u5efa\u8def\u7531\u7ec4",level:3},{value:"\u7ec4\u7ea7\u4e2d\u95f4\u4ef6",id:"\u7ec4\u7ea7\u4e2d\u95f4\u4ef6",level:3},{value:"\u5d4c\u5957\u8def\u7531\u7ec4",id:"\u5d4c\u5957\u8def\u7531\u7ec4",level:3},{value:"\u8def\u7531\u53c2\u6570",id:"\u8def\u7531\u53c2\u6570",level:2},{value:"\u53c2\u6570\u8def\u7531",id:"\u53c2\u6570\u8def\u7531",level:3},{value:"\u6b63\u5219\u8def\u7531\u53c2\u6570",id:"\u6b63\u5219\u8def\u7531\u53c2\u6570",level:3},{value:"\u901a\u914d\u7b26\u8def\u7531",id:"\u901a\u914d\u7b26\u8def\u7531",level:3},{value:"\u8def\u7531\u5339\u914d\u4f18\u5148\u7ea7",id:"\u8def\u7531\u5339\u914d\u4f18\u5148\u7ea7",level:3},{value:"\u9759\u6001\u8d44\u6e90\u8def\u7531",id:"\u9759\u6001\u8d44\u6e90\u8def\u7531",level:2},{value:"\u57fa\u672c\u7528\u6cd5",id:"\u57fa\u672c\u7528\u6cd5",level:3},{value:"\u9ad8\u7ea7\u914d\u7f6e",id:"\u9ad8\u7ea7\u914d\u7f6e",level:3},{value:"\u6587\u4ef6\u4e0a\u4f20\u548c\u4e0b\u8f7d",id:"\u6587\u4ef6\u4e0a\u4f20\u548c\u4e0b\u8f7d",level:3},{value:"\u7efc\u5408\u793a\u4f8b",id:"\u7efc\u5408\u793a\u4f8b",level:2}];function u(r){const e={code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",ol:"ol",p:"p",pre:"pre",strong:"strong",...(0,a.R)(),...r.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(e.header,{children:(0,s.jsx)(e.h1,{id:"router",children:"Router"})}),"\n",(0,s.jsx)(e.p,{children:"WebFrame \u652f\u6301\u591a\u79cd\u8def\u7531\u6a21\u5f0f\u3001\u8def\u7531\u5206\u7ec4\u3001\u53c2\u6570\u63d0\u53d6\u548c\u9759\u6001\u8d44\u6e90\u670d\u52a1\u3002"}),"\n",(0,s.jsx)(e.h2,{id:"\u8def\u7531\u6ce8\u518c",children:"\u8def\u7531\u6ce8\u518c"}),"\n",(0,s.jsx)(e.h3,{id:"\u57fa\u7840\u8def\u7531\u6ce8\u518c",children:"\u57fa\u7840\u8def\u7531\u6ce8\u518c"}),"\n",(0,s.jsx)(e.p,{children:"WebFrame \u652f\u6301\u6240\u6709\u6807\u51c6\u7684 HTTP \u65b9\u6cd5\uff0c\u901a\u8fc7\u7b80\u5355\u7684 API \u8fdb\u884c\u8def\u7531\u6ce8\u518c\uff1a"}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'func main() {\r\n    server := web.NewHTTPServer()\r\n    \r\n    // \u6ce8\u518c GET \u8def\u7531\r\n    server.Get("/hello", func(ctx *web.Context) {\r\n        ctx.String(200, "Hello World!")\r\n    })\r\n    \r\n    // \u6ce8\u518c POST \u8def\u7531\r\n    server.Post("/users", func(ctx *web.Context) {\r\n        // \u5904\u7406\u521b\u5efa\u7528\u6237\r\n        ctx.JSON(201, map[string]string{"id": "123", "name": "new user"})\r\n    })\r\n    \r\n    // \u6ce8\u518c PUT \u8def\u7531\r\n    server.Put("/users/:id", func(ctx *web.Context) {\r\n        id := ctx.PathParam("id").Value\r\n        ctx.String(200, "update user: "+id)\r\n    })\r\n    \r\n    // \u6ce8\u518c DELETE \u8def\u7531\r\n    server.Delete("/users/:id", func(ctx *web.Context) {\r\n        id := ctx.PathParam("id").Value\r\n        ctx.String(200, "delete user: "+id)\r\n    })\r\n    \r\n    // \u6ce8\u518c PATCH \u8def\u7531\r\n    server.Patch("/users/:id/status", func(ctx *web.Context) {\r\n        id := ctx.PathParam("id").Value\r\n        ctx.String(200, "update user status: "+id)\r\n    })\r\n    \r\n    // \u6ce8\u518c OPTIONS \u8def\u7531\r\n    server.Options("/users", func(ctx *web.Context) {\r\n        ctx.Resp.Header().Set("Allow", "GET, POST, PUT, DELETE")\r\n        ctx.Status(204)\r\n    })\r\n    \r\n    server.Start(":8080")\r\n}\n'})}),"\n",(0,s.jsx)(e.h3,{id:"\u94fe\u5f0f-api",children:"\u94fe\u5f0f API"}),"\n",(0,s.jsx)(e.p,{children:"WebFrame \u652f\u6301\u94fe\u5f0f API \u98ce\u683c\uff0c\u53ef\u4ee5\u5728\u8def\u7531\u6ce8\u518c\u540e\u76f4\u63a5\u6dfb\u52a0\u4e2d\u95f4\u4ef6\uff1a"}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'server.Get("/admin/dashboard", func(ctx *web.Context) {\r\n    ctx.String(200, "admin dashbord")\r\n}).Middleware(\r\n    // \u6dfb\u52a0\u8ba4\u8bc1\u4e2d\u95f4\u4ef6\r\n    func(next web.HandlerFunc) web.HandlerFunc {\r\n        return func(ctx *web.Context) {\r\n            // \u9a8c\u8bc1\u6743\u9650\r\n            if !isAdmin(ctx) {\r\n                ctx.String(403, "access denied")\r\n                return\r\n            }\r\n            next(ctx)\r\n        }\r\n    },\r\n    // \u6dfb\u52a0\u65e5\u5fd7\u4e2d\u95f4\u4ef6\r\n    func(next web.HandlerFunc) web.HandlerFunc {\r\n        return func(ctx *web.Context) {\r\n            fmt.Println("access admin dashboard")\r\n            next(ctx)\r\n        }\r\n    },\r\n)\n'})}),"\n",(0,s.jsx)(e.h2,{id:"\u8def\u7531\u7ec4",children:"\u8def\u7531\u7ec4"}),"\n",(0,s.jsx)(e.p,{children:"\u8def\u7531\u7ec4\u5141\u8bb8\u60a8\u5c06\u76f8\u5173\u8def\u7531\u7ec4\u7ec7\u5728\u4e00\u8d77\uff0c\u5171\u4eab\u516c\u5171\u524d\u7f00\u548c\u4e2d\u95f4\u4ef6\u3002"}),"\n",(0,s.jsx)(e.h3,{id:"\u521b\u5efa\u8def\u7531\u7ec4",children:"\u521b\u5efa\u8def\u7531\u7ec4"}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'func main() {\r\n    server := web.NewHTTPServer()\r\n    \r\n    // \u521b\u5efa API \u8def\u7531\u7ec4\r\n    api := server.Group("/api")\r\n    \r\n    // \u6ce8\u518c API \u8def\u7531\r\n    api.Get("/users", listUsers)\r\n    api.Post("/users", createUser)\r\n    \r\n    // \u521b\u5efa v1 \u7248\u672c API \u5b50\u7ec4\r\n    v1 := api.Group("/v1")\r\n    v1.Get("/products", listProductsV1)\r\n    \r\n    // \u521b\u5efa v2 \u7248\u672c API \u5b50\u7ec4\r\n    v2 := api.Group("/v2")\r\n    v2.Get("/products", listProductsV2)\r\n    \r\n    server.Start(":8080")\r\n}\n'})}),"\n",(0,s.jsx)(e.h3,{id:"\u7ec4\u7ea7\u4e2d\u95f4\u4ef6",children:"\u7ec4\u7ea7\u4e2d\u95f4\u4ef6"}),"\n",(0,s.jsx)(e.p,{children:"\u53ef\u4ee5\u4e3a\u6574\u4e2a\u8def\u7531\u7ec4\u6dfb\u52a0\u4e2d\u95f4\u4ef6\uff0c\u8fd9\u4e9b\u4e2d\u95f4\u4ef6\u5c06\u5e94\u7528\u4e8e\u8be5\u7ec4\u4e2d\u7684\u6240\u6709\u8def\u7531\uff1a"}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'// \u521b\u5efa\u8def\u7531\u7ec4\u5e76\u6dfb\u52a0\u4e2d\u95f4\u4ef6\r\nusersGroup := server.Group("/users").Use(\r\n    authMiddleware,\r\n    loggingMiddleware,\r\n)\r\n\r\n// \u7ec4\u4e2d\u7684\u6240\u6709\u8def\u7531\u90fd\u4f1a\u5e94\u7528\u4e0a\u9762\u7684\u4e2d\u95f4\u4ef6\r\nusersGroup.Get("", listUsers)\r\nusersGroup.Get("/:id", getUserById)\r\nusersGroup.Post("", createUser)\n'})}),"\n",(0,s.jsx)(e.h3,{id:"\u5d4c\u5957\u8def\u7531\u7ec4",children:"\u5d4c\u5957\u8def\u7531\u7ec4"}),"\n",(0,s.jsx)(e.p,{children:"\u8def\u7531\u7ec4\u53ef\u4ee5\u65e0\u9650\u5d4c\u5957\uff0c\u6bcf\u4e2a\u5b50\u7ec4\u7ee7\u627f\u7236\u7ec4\u7684\u8def\u5f84\u524d\u7f00\u548c\u4e2d\u95f4\u4ef6\uff1a"}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'// \u4e3b API \u7ec4\r\napi := server.Group("/api")\r\n\r\n// \u8ba4\u8bc1 API \u5b50\u7ec4\r\nauth := api.Group("/auth")\r\nauth.Post("/login", handleLogin)\r\nauth.Post("/register", handleRegister)\r\n\r\n// \u7528\u6237 API \u5b50\u7ec4\r\nusers := api.Group("/users").Use(authRequired)\r\nusers.Get("", listUsers)\r\n\r\n// \u7528\u6237\u6587\u6863\u5b50\u7ec4\r\nuserDocs := users.Group("/:id/documents")\r\nuserDocs.Get("", listUserDocuments)\r\nuserDocs.Post("", uploadUserDocument)\n'})}),"\n",(0,s.jsx)(e.h2,{id:"\u8def\u7531\u53c2\u6570",children:"\u8def\u7531\u53c2\u6570"}),"\n",(0,s.jsx)(e.p,{children:"WebFrame \u652f\u6301\u591a\u79cd\u7c7b\u578b\u7684\u8def\u7531\u53c2\u6570\uff0c\u80fd\u591f\u6ee1\u8db3\u5404\u79cd\u590d\u6742\u7684 URL \u5339\u914d\u9700\u6c42\u3002"}),"\n",(0,s.jsx)(e.h3,{id:"\u53c2\u6570\u8def\u7531",children:"\u53c2\u6570\u8def\u7531"}),"\n",(0,s.jsxs)(e.p,{children:["\u4f7f\u7528 ",(0,s.jsx)(e.code,{children:":param"})," \u8bed\u6cd5\u5b9a\u4e49\u8def\u5f84\u53c2\u6570\uff1a"]}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'server.Get("/users/:id", func(ctx *web.Context) {\r\n    id := ctx.PathParam("id").Value\r\n    ctx.String(200, "\u7528\u6237 ID: "+id)\r\n})\r\n\r\nserver.Get("/blogs/:year/:month/:day/:slug", func(ctx *web.Context) {\r\n    year := ctx.PathParam("year").Value\r\n    month := ctx.PathParam("month").Value\r\n    day := ctx.PathParam("day").Value\r\n    slug := ctx.PathParam("slug").Value\r\n    \r\n    ctx.JSON(200, map[string]string{\r\n        "year": year,\r\n        "month": month,\r\n        "day": day,\r\n        "slug": slug,\r\n    })\r\n})\n'})}),"\n",(0,s.jsxs)(e.p,{children:["\u53c2\u6570\u503c\u53ef\u4ee5\u901a\u8fc7 ",(0,s.jsx)(e.code,{children:"Context"})," \u5bf9\u8c61\u7684\u65b9\u6cd5\u83b7\u53d6\u5e76\u8f6c\u6362\u4e3a\u6240\u9700\u7c7b\u578b\uff1a"]}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'server.Get("/products/:id/reviews/:score", func(ctx *web.Context) {\r\n    // \u83b7\u53d6\u5b57\u7b26\u4e32\u53c2\u6570\r\n    idStr := ctx.PathParam("id").Value\r\n    \r\n    // \u83b7\u53d6\u5e76\u8f6c\u6362\u4e3a\u6574\u6570\r\n    id := ctx.PathInt("id").Value\r\n    \r\n    // \u83b7\u53d6\u5e76\u8f6c\u6362\u4e3a\u6d6e\u70b9\u6570\r\n    score := ctx.PathFloat("score").Value\r\n    \r\n    ctx.JSON(200, map[string]interface{}{\r\n        "product_id": id,\r\n        "score": score,\r\n    })\r\n})\n'})}),"\n",(0,s.jsx)(e.h3,{id:"\u6b63\u5219\u8def\u7531\u53c2\u6570",children:"\u6b63\u5219\u8def\u7531\u53c2\u6570"}),"\n",(0,s.jsx)(e.p,{children:"\u53ef\u4ee5\u4f7f\u7528\u6b63\u5219\u8868\u8fbe\u5f0f\u9650\u5236\u53c2\u6570\u683c\u5f0f\uff1a"}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'// \u9650\u5236 ID \u53ea\u80fd\u662f\u6570\u5b57\r\nserver.Get("/users/:id([0-9]+)", func(ctx *web.Context) {\r\n    id := ctx.PathParam("id").Value\r\n    ctx.String(200, "\u7528\u6237 ID (\u6570\u5b57): "+id)\r\n})\r\n\r\n// \u9650\u5236\u7528\u6237\u540d\u53ea\u80fd\u662f\u5b57\u6bcd\r\nserver.Get("/users/:username([a-zA-Z]+)", func(ctx *web.Context) {\r\n    username := ctx.PathParam("username").Value\r\n    ctx.String(200, "\u7528\u6237\u540d (\u5b57\u6bcd): "+username)\r\n})\r\n\r\n// \u66f4\u590d\u6742\u7684\u6b63\u5219\u8868\u8fbe\u5f0f\r\nserver.Get("/articles/:slug([a-z0-9-]+)", func(ctx *web.Context) {\r\n    slug := ctx.PathParam("slug").Value\r\n    ctx.String(200, "\u6587\u7ae0 Slug: "+slug)\r\n})\n'})}),"\n",(0,s.jsx)(e.h3,{id:"\u901a\u914d\u7b26\u8def\u7531",children:"\u901a\u914d\u7b26\u8def\u7531"}),"\n",(0,s.jsxs)(e.p,{children:["\u4f7f\u7528 ",(0,s.jsx)(e.code,{children:"*"})," \u5339\u914d\u4efb\u610f\u8def\u5f84\u6bb5\uff1a"]}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'// \u5339\u914d /files/ \u540e\u7684\u4efb\u4f55\u8def\u5f84\r\nserver.Get("/files/*", func(ctx *web.Context) {\r\n    path := ctx.PathParam("file").Value\r\n    ctx.String(200, "\u8bf7\u6c42\u7684\u6587\u4ef6\u8def\u5f84: "+path)\r\n})\r\n\r\n// \u5904\u7406\u6240\u6709\u672a\u627e\u5230\u7684\u8def\u7531\r\nserver.Get("/*", func(ctx *web.Context) {\r\n    ctx.String(404, "\u672a\u627e\u5230\u9875\u9762")\r\n})\n'})}),"\n",(0,s.jsx)(e.h3,{id:"\u8def\u7531\u5339\u914d\u4f18\u5148\u7ea7",children:"\u8def\u7531\u5339\u914d\u4f18\u5148\u7ea7"}),"\n",(0,s.jsx)(e.p,{children:"WebFrame \u8def\u7531\u5339\u914d\u9075\u5faa\u4ee5\u4e0b\u4f18\u5148\u7ea7\u89c4\u5219\uff1a"}),"\n",(0,s.jsxs)(e.ol,{children:["\n",(0,s.jsxs)(e.li,{children:[(0,s.jsx)(e.strong,{children:"\u9759\u6001\u8def\u7531"}),"\uff1a\u5b8c\u5168\u5339\u914d\u7684\u8def\u5f84\uff0c\u5982 ",(0,s.jsx)(e.code,{children:"/users/profile"})]}),"\n",(0,s.jsxs)(e.li,{children:[(0,s.jsx)(e.strong,{children:"\u6b63\u5219\u8def\u7531"}),"\uff1a\u5305\u542b\u6b63\u5219\u8868\u8fbe\u5f0f\u7684\u53c2\u6570\u8def\u5f84\uff0c\u5982 ",(0,s.jsx)(e.code,{children:"/users/:id([0-9]+)"})]}),"\n",(0,s.jsxs)(e.li,{children:[(0,s.jsx)(e.strong,{children:"\u53c2\u6570\u8def\u7531"}),"\uff1a\u5305\u542b\u53c2\u6570\u7684\u8def\u5f84\uff0c\u5982 ",(0,s.jsx)(e.code,{children:"/users/:id"})]}),"\n",(0,s.jsxs)(e.li,{children:[(0,s.jsx)(e.strong,{children:"\u901a\u914d\u7b26\u8def\u7531"}),"\uff1a\u5305\u542b\u901a\u914d\u7b26\u7684\u8def\u5f84\uff0c\u5982 ",(0,s.jsx)(e.code,{children:"/users/*"})]}),"\n"]}),"\n",(0,s.jsxs)(e.p,{children:["\u4f8b\u5982\uff0c\u5bf9\u4e8e\u8bf7\u6c42 ",(0,s.jsx)(e.code,{children:"/users/123"}),"\uff0c\u5339\u914d\u987a\u5e8f\u4e3a\uff1a"]}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'server.Get("/users/123", func(ctx *web.Context) {\r\n    // 1. \u9996\u5148\u5339\u914d\u8fd9\u4e2a\u9759\u6001\u8def\u7531\r\n})\r\n\r\nserver.Get("/users/:id([0-9]+)", func(ctx *web.Context) {\r\n    // 2. \u5176\u6b21\u5339\u914d\u8fd9\u4e2a\u6b63\u5219\u8def\u7531\r\n})\r\n\r\nserver.Get("/users/:id", func(ctx *web.Context) {\r\n    // 3. \u7136\u540e\u5339\u914d\u8fd9\u4e2a\u53c2\u6570\u8def\u7531\r\n})\r\n\r\nserver.Get("/users/*", func(ctx *web.Context) {\r\n    // 4. \u6700\u540e\u5339\u914d\u8fd9\u4e2a\u901a\u914d\u7b26\u8def\u7531\r\n})\n'})}),"\n",(0,s.jsx)(e.h2,{id:"\u9759\u6001\u8d44\u6e90\u8def\u7531",children:"\u9759\u6001\u8d44\u6e90\u8def\u7531"}),"\n",(0,s.jsx)(e.p,{children:"WebFrame \u63d0\u4f9b\u4e86\u5185\u7f6e\u652f\u6301\uff0c\u7528\u4e8e\u670d\u52a1\u9759\u6001\u6587\u4ef6\uff0c\u5982 CSS\u3001JavaScript\u3001\u56fe\u7247\u7b49\u3002"}),"\n",(0,s.jsx)(e.h3,{id:"\u57fa\u672c\u7528\u6cd5",children:"\u57fa\u672c\u7528\u6cd5"}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'func main() {\r\n    server := web.NewHTTPServer()\r\n    \r\n    // \u521b\u5efa\u9759\u6001\u8d44\u6e90\u5904\u7406\u5668\r\n    staticResource := web.NewStaticResource("./static")\r\n    \r\n    // \u6ce8\u518c\u9759\u6001\u8d44\u6e90\u8def\u7531\r\n    server.Use("GET", "/static/*", staticResource.Handle())\r\n    \r\n    // \u542f\u52a8\u670d\u52a1\u5668\r\n    server.Start(":8080")\r\n}\n'})}),"\n",(0,s.jsx)(e.h3,{id:"\u9ad8\u7ea7\u914d\u7f6e",children:"\u9ad8\u7ea7\u914d\u7f6e"}),"\n",(0,s.jsx)(e.p,{children:"\u9759\u6001\u8d44\u6e90\u5904\u7406\u5668\u652f\u6301\u591a\u79cd\u914d\u7f6e\u9009\u9879\uff1a"}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'func main() {\r\n    server := web.NewHTTPServer()\r\n    \r\n    // \u521b\u5efa\u5e26\u914d\u7f6e\u7684\u9759\u6001\u8d44\u6e90\u5904\u7406\u5668\r\n    staticResource := web.NewStaticResource(\r\n        "./public",\r\n        web.WithPathPrefix("/assets/"),\r\n        web.WithMaxSize(10 << 20), // 10MB \u7f13\u5b58\u9650\u5236\r\n        web.WithCache(time.Hour, 10*time.Minute), // \u7f13\u5b58\u914d\u7f6e\r\n        web.WithExtContentTypes(map[string]string{\r\n            ".css":  "text/css; charset=utf-8",\r\n            ".js":   "application/javascript",\r\n            ".png":  "image/png",\r\n            ".jpg":  "image/jpeg",\r\n            ".jpeg": "image/jpeg",\r\n            ".gif":  "image/gif",\r\n            ".svg":  "image/svg+xml",\r\n            ".woff": "font/woff",\r\n            ".woff2": "font/woff2",\r\n        }),\r\n    )\r\n    \r\n    // \u6ce8\u518c\u9759\u6001\u8d44\u6e90\u8def\u7531\r\n    server.Use("GET", "/assets/*", staticResource.Handle())\r\n    \r\n    server.Start(":8080")\r\n}\n'})}),"\n",(0,s.jsx)(e.h3,{id:"\u6587\u4ef6\u4e0a\u4f20\u548c\u4e0b\u8f7d",children:"\u6587\u4ef6\u4e0a\u4f20\u548c\u4e0b\u8f7d"}),"\n",(0,s.jsx)(e.p,{children:"WebFrame \u4e5f\u63d0\u4f9b\u4e86\u6587\u4ef6\u4e0a\u4f20\u548c\u4e0b\u8f7d\u7684\u5904\u7406\u5668\uff1a"}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'func main() {\r\n    server := web.NewHTTPServer()\r\n    \r\n    // \u6587\u4ef6\u4e0a\u4f20\u5904\u7406\r\n    uploader := web.NewFileUploader(\r\n        "upload_file", // \u8868\u5355\u5b57\u6bb5\u540d\r\n        "./uploads",   // \u4e0a\u4f20\u76ee\u6807\u76ee\u5f55\r\n        web.WithFileMaxSize(50 << 20), // \u9650\u5236 50MB\r\n        web.WithAllowedTypes([]string{ // \u9650\u5236\u6587\u4ef6\u7c7b\u578b\r\n            "image/jpeg",\r\n            "image/png",\r\n            "application/pdf",\r\n        }),\r\n    )\r\n    server.Post("/upload", uploader.HandleUpload())\r\n    \r\n    // \u6587\u4ef6\u4e0b\u8f7d\u5904\u7406\r\n    downloader := web.FileDownloader{\r\n        DestPath: "./downloads",\r\n    }\r\n    server.Get("/download/:file", downloader.HandleDownload())\r\n    \r\n    server.Start(":8080")\r\n}\n'})}),"\n",(0,s.jsx)(e.h2,{id:"\u7efc\u5408\u793a\u4f8b",children:"\u7efc\u5408\u793a\u4f8b"}),"\n",(0,s.jsx)(e.p,{children:"\u4e0b\u9762\u662f\u4e00\u4e2a\u7efc\u5408\u4f7f\u7528\u8def\u7531\u7cfb\u7edf\u5404\u79cd\u529f\u80fd\u7684\u5b8c\u6574\u793a\u4f8b\uff1a"}),"\n",(0,s.jsx)(e.pre,{children:(0,s.jsx)(e.code,{className:"language-go",children:'package main\r\n\r\nimport (\r\n    "fmt"\r\n    "github.com/fyerfyer/fyer-webframe/web"\r\n    "github.com/fyerfyer/fyer-webframe/web/middleware/accesslog"\r\n    "github.com/fyerfyer/fyer-webframe/web/middleware/recovery"\r\n    "net/http"\r\n    "time"\r\n)\r\n\r\nfunc main() {\r\n    // \u521b\u5efa\u670d\u52a1\u5668\r\n    server := web.NewHTTPServer()\r\n    \r\n    // \u6dfb\u52a0\u5168\u5c40\u4e2d\u95f4\u4ef6\r\n    server.Use("*", "*", recovery.Recovery())\r\n    server.Use("*", "*", accesslog.NewMiddlewareBuilder().Build())\r\n    \r\n    // \u9759\u6001\u8d44\u6e90\u914d\u7f6e\r\n    staticFiles := web.NewStaticResource(\r\n        "./public",\r\n        web.WithPathPrefix("/static/"),\r\n        web.WithCache(time.Hour, 10*time.Minute),\r\n    )\r\n    server.Use("GET", "/static/*", staticFiles.Handle())\r\n    \r\n    // \u57fa\u672c\u8def\u7531\r\n    server.Get("/", func(ctx *web.Context) {\r\n        ctx.HTML(200, "<h1>\u6b22\u8fce\u4f7f\u7528 WebFrame</h1>")\r\n    })\r\n    \r\n    // API \u8def\u7531\u7ec4\r\n    api := server.Group("/api")\r\n    \r\n    // v1 API \u7248\u672c\r\n    v1 := api.Group("/v1")\r\n    \r\n    // \u8ba4\u8bc1\u5b50\u7ec4\r\n    auth := v1.Group("/auth")\r\n    auth.Post("/login", handleLogin)\r\n    auth.Post("/register", handleRegister)\r\n    \r\n    // \u7528\u6237\u5b50\u7ec4 (\u9700\u8981\u8ba4\u8bc1)\r\n    users := v1.Group("/users").Use(authMiddleware)\r\n    users.Get("", listUsers)\r\n    users.Get("/:id([0-9]+)", getUserById)\r\n    users.Put("/:id([0-9]+)", updateUser)\r\n    users.Delete("/:id([0-9]+)", deleteUser)\r\n    \r\n    // \u7528\u6237\u6587\u6863\u5b50\u7ec4\r\n    docs := users.Group("/:user_id([0-9]+)/documents")\r\n    docs.Get("", listUserDocuments)\r\n    docs.Get("/:doc_id", getUserDocument)\r\n    docs.Post("", uploadUserDocument)\r\n    \r\n    fmt.Println("\u670d\u52a1\u5668\u542f\u52a8\u5728 :8080")\r\n    server.Start(":8080")\r\n}\r\n\r\n// \u5904\u7406\u51fd\u6570\r\nfunc handleLogin(ctx *web.Context) {\r\n    // \u767b\u5f55\u903b\u8f91\r\n}\r\n\r\nfunc handleRegister(ctx *web.Context) {\r\n    // \u6ce8\u518c\u903b\u8f91\r\n}\r\n\r\nfunc listUsers(ctx *web.Context) {\r\n    // \u5217\u51fa\u7528\u6237\r\n    ctx.JSON(200, []map[string]interface{}{\r\n        {"id": 1, "name": "\u7528\u62371"},\r\n        {"id": 2, "name": "\u7528\u62372"},\r\n    })\r\n}\r\n\r\nfunc getUserById(ctx *web.Context) {\r\n    id := ctx.PathInt("id").Value\r\n    ctx.JSON(200, map[string]interface{}{\r\n        "id": id,\r\n        "name": fmt.Sprintf("\u7528\u6237%d", id),\r\n    })\r\n}\r\n\r\nfunc updateUser(ctx *web.Context) {\r\n    // \u66f4\u65b0\u7528\u6237\r\n}\r\n\r\nfunc deleteUser(ctx *web.Context) {\r\n    // \u5220\u9664\u7528\u6237\r\n}\r\n\r\nfunc listUserDocuments(ctx *web.Context) {\r\n    userId := ctx.PathInt("user_id").Value\r\n    ctx.JSON(200, map[string]interface{}{\r\n        "user_id": userId,\r\n        "documents": []map[string]interface{}{\r\n            {"id": 1, "name": "\u6587\u68631"},\r\n            {"id": 2, "name": "\u6587\u68632"},\r\n        },\r\n    })\r\n}\r\n\r\nfunc getUserDocument(ctx *web.Context) {\r\n    userId := ctx.PathInt("user_id").Value\r\n    docId := ctx.PathParam("doc_id").Value\r\n    ctx.JSON(200, map[string]interface{}{\r\n        "user_id": userId,\r\n        "doc_id": docId,\r\n        "name": "\u7528\u6237\u6587\u6863",\r\n    })\r\n}\r\n\r\nfunc uploadUserDocument(ctx *web.Context) {\r\n    // \u4e0a\u4f20\u6587\u6863\u903b\u8f91\r\n}\r\n\r\n// \u4e2d\u95f4\u4ef6\r\nfunc authMiddleware(next web.HandlerFunc) web.HandlerFunc {\r\n    return func(ctx *web.Context) {\r\n        token := ctx.GetHeader("Authorization")\r\n        if token == "" {\r\n            ctx.JSON(http.StatusUnauthorized, map[string]string{\r\n                "error": "\u672a\u6388\u6743\u8bbf\u95ee",\r\n            })\r\n            return\r\n        }\r\n        // \u5728\u5b9e\u9645\u5e94\u7528\u4e2d\u9a8c\u8bc1\u4ee4\u724c\r\n        next(ctx)\r\n    }\r\n}\n'})})]})}function o(r={}){const{wrapper:e}={...(0,a.R)(),...r.components};return e?(0,s.jsx)(e,{...r,children:(0,s.jsx)(u,{...r})}):u(r)}},8453:(r,e,n)=>{n.d(e,{R:()=>i,x:()=>c});var t=n(6540);const s={},a=t.createContext(s);function i(r){const e=t.useContext(a);return t.useMemo((function(){return"function"==typeof r?r(e):{...e,...r}}),[e,r])}function c(r){let e;return e=r.disableParentContext?"function"==typeof r.components?r.components(s):r.components||s:i(r.components),t.createElement(a.Provider,{value:e},r.children)}}}]);