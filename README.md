# MCP Rajomon 治理网关

一个基于 Rajomon 治理思想的 AI 服务（MCP 协议）智能网关，实现动态成本定价和流量控制。

## 🎯 项目概述

本项目将 Rajomon 的服务治理思想应用于 AI 服务场景，为 MCP（Model Context Protocol）协议的服务提供动态定价、流量控制和可观测性能力。

### 核心特性

- **动态定价算法**: 基于 EWMA（指数加权移动平均）延迟和 Token 消耗的动态定价
- **MCP 协议支持**: 完整支持 MCP 协议的模拟服务端实现
- **智能准入控制**: 基于客户端 Token 余额的请求过滤
- **可观测性集成**: Prometheus 指标暴露和 Grafana 监控
- **云原生部署**: 完整的 Docker 容器化部署方案
