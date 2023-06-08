# What is AlterShield Operator
## Introduction
The positioning of Operator is to serve as a perception and expansion of AlterShield in the cloud-native field of change, and our goal is to create a universal [Kubernetes Custom Resource (CRD)][CRD] control framework for change prevention.

AlterShield Operator is a Kubernetes Operator developed based on [Operator-SKD][Operator-SKD], aimed at improving the stability and reliability of Kubernetes clusters through control of Workload resources. This makes AlterShield not only suitable for traditional SOA architecture but also well-suited for cloud-native microservice system design.

You can also deploy Operator independently from the main [AlterShield][AlterShield] product, quickly deploying it to your Kubernetes cluster. Currently, Operator has achieved awareness of changes to Deployments, change prevention, change interception, change self-healing, and rollback. 
More Workload controls and additional release methods are under development.

## Design
In Kubernetes, [Pod][Pod] is the smallest deployment unit. Pods are typically used to host a group of related containers that work together to accomplish a specific task. Workload is an abstract layer for managing application deployments in Kubernetes. It helps users deploy, manage, and monitor applications in a Kubernetes cluster. Workloads include [Deployment][Deployment], [StatefulSet][StatefulSet], [DaemonSet][DaemonSet], [Job][Job] and [CronJob][CronJob], which define different ways and behaviors for deploying applications.

In robotics and automation, a control loop is a non-terminating loop that regulates the system's status. Users can also define their own resources and customize the coordination logic of their resources. In line with Kubernetes' design philosophy, we have defined two CRDs, namely [ChangeWorkload][Change WorkLoad] and [ChangePod][Change Pod], which primarily describe and define changes while tracking the entire change lifecycle.

The lifecycle is as follows:

![img.png](docs/img.png)

The basic idea is to use the [WebHook][WebHook] capability of the Kubernetes API server to detect updates to CRDs. We typically define updates to WorkLoads as a type of change, with the minimum unit of change corresponding to the minimum scheduling unit of Kubernetes, which is the Pod. This design philosophy is consistent with the change information model defined by AlterShield. Through CRDs, we can declaratively describe and define changes, and the Operator internally implements the go language version of the AlterShield client, which accesses AlterShield according to the standard [change information model][ChangeModel].


## Features
The AlterShield Operator currently offers the following capabilities:

1、Admission control, which controls the deployment of Deployment resources to prevent deployment of resources under abnormal conditions.

2、Runtime monitoring, which monitors the PODs under Deployment resources to ensure the running status of resources.

3、Automatic rollback and self-healing, which automatically rolls back the abnormal states of Deployment resources and self-heals the faulty releases. Through these features, the AlterShield Operator can help users better manage Workload resources in their Kubernetes clusters, thus improving system stability and reliability.

## RoadMap
1、Stage one (completed)
- [x] Operator quick experience with native Deployment change awareness, change defense, change interception, and change self-healing (rollback) in K8S cluster.
- [x] Built-in AlterShield SDK standard G2 scenario access in Go language.

2、Stage two
- [ ] Abstract the CRD control framework to support quick access to custom CRD change awareness.
- [ ] Abstract the dependence on AlterShield to support completely independent use of Operator application change control framework.
- [ ] Provide plug-in validation service capability to support declarative validation rule writing.

3、Stage three
- [ ] Support intelligent algorithmic defense and validation services that can be used out-of-the-box with the integration of cloud-native monitoring components.


## Contribute to Operator

If you are experiencing pain points such as inability to perceive, prevent, or self-heal changes in a cloud-native environment, we welcome your participation in building Operator together.
We hope that the defined [change information model][ChangeModel] is also applicable in the cloud-native field, but this project is more about exploring how changes in the cloud-native community can be defined and what kind of change prevention and control technology framework is more cloud-native. The above RoadMap describes our exploration plan, and we hope to find more members from the cloud-native community, SRE community, and stability community to work together to improve a more general change information model and CRD prevention and control framework.
Ways you may participate include:

1、Define your own CRD webhook and convert it to the built-in CRD change information model.

2、Access and extend open-source cloud-native monitoring components such as [HoloInsight][HoloInsight], [Prometheus][Prometheus].

3、Participate in the scheme discussion and implementation of CRD prevention and control framework.

4、Participate in the regularization.

<!-- LICENSE -->
<!-- LICENSE -->
## License

Distributed under the Apache2.0 License. See `LICENSE` for more information.


<!-- CONTACT -->
## Contact
- Contact us via Email: traas_stack@antgroup.com / altershield.io@gmail.com
- Ding Talk Group [QR code](./docs/dingtalk.png)
- WeChat Official Account [QR code](./docs/wechat.jpg)
- <a href="https://altershield.slack.com/"><img src="https://img.shields.io/badge/slack-AlterShield-0abd59?logo=slack" alt="slack" /></a>


[AlterShield]:https://github.com/traas-stack/altershield
[Change WorkLoad]:https://github.com/traas-stack/altershield-operator/blob/main/apis/app.ops.cloud.alipay.com/v1alpha1/changeworkload_types.go
[Change Pod]:https://github.com/traas-stack/altershield-operator/blob/main/apis/app.ops.cloud.alipay.com/v1alpha1/changepod_types.go
[ChangeModel]:https://traas-stack.github.io/altershield-docs/zh-CN/open-change-management-specification/change-model/
[Pod]:https://kubernetes.io/docs/concepts/workloads/pods/
[Operator-SKD]:https://sdk.operatorframework.io/
[CRD]:https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
[Deployment]:https://kubernetes.io/docs/concepts/workloads/controllers/deployment/
[WebHook]:https://kubernetes.io/docs/reference/access-authn-authz/
[StatefulSet]:https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/
[DaemonSet]:https://kubernetes.io/docs/concepts/workloads/controllers/daemonset/
[Job]:https://kubernetes.io/docs/concepts/workloads/controllers/job/
[CronJob]:https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/
[HoloInsight]:https://github.com/traas-stack/holoinsight
[Prometheus]:https://prometheus.io/
