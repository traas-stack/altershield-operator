# AlterShield Operator是什么
## 简介
AlterShield Operator是一款基于Operator-SKD开发的Kubernetes Operator，旨在通过对Workload资源的管控，提高Kubernetes集群的稳定性和可靠性。目前，AlterShield Operator支持对Deployment资源进行管控，而其他通用资源正在不断开发中。其核心设计目标是开发迅速、学习简单、易于扩展。

## 功能
AlterShield Operator具有以下主要功能：

1. 准入控制，对Workload资源的部署进行管控，防止异常状态下的资源部署

2. 运行检测，对Workload资源下的POD进行监测，确保资源的运行状态

3. 回滚自愈，对Workload资源的异常状态进行自动回滚，自愈回滚异常的发布

通过这些功能，AlterShield Operator可以帮助用户更好地管理Kubernetes集群中的Workload资源，提高系统的稳定性和可靠性。
