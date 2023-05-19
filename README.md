# What is AlterShield Operator
## Overview
AlterShield Operator is a Kubernetes Operator developed based on Operator-SKD. Its purpose is to improve the stability and reliability of Kubernetes clusters by controlling Workload resources. Currently, AlterShield Operator supports controlling Deployment resources, while other common resources are continuously being developed. Its core design goals are rapid development, easy learning, and easy scalability.

## Features
AlterShield Operator has the following main functions:

1. Admission control, controlling the deployment of Workload resources to prevent deployment under abnormal conditions

2. Run detection, monitoring the PODs under Workload resources to ensure the running status of resources

3. Rollback self-healing, automatically rolling back the abnormal state of Workload resources and self-healing the rollback exception

Through these functions, AlterShield Operator can help users better manage Workload resources in Kubernetes clusters and improve system stability and reliability.
