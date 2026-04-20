# gocommon
go公共组件：鉴权，三方登录，支付，OSS，等。

- auth
  - 三方登录的支持，再补充github登录，google登录。
- logger
  - 采用zap的logger，要求kratos可以直接使用，
- middleware
  - 常用中间件
    - 1. 统一接口请求成功/错误返回
    - 2. auth鉴权拦截
- oss
- redis
- utils
  - pay
  - sms
