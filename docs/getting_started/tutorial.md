# GLBC

The Global Load Balancer Controller (GLBC) leverages [`kcp`](https://github.com/Kuadrant/kcp) to provide DNS-based global load balancing and transparent multi-cluster ingress. The main API for the GLBC is the Kubernetes Ingress object. GLBC watches Ingress objects and transforms them adding in the GLBC managed host and TLS certificate.

For more information on the architecture of GLBC and how the various component work, refer to the [overview documentation](/docs/getting_started/overview.md).

Use this tutorial to perform the following actions:

* Install the kcp-glbc instance and verify installation.
* Follow the demo and have GLBC running and working with an AWS domain. You can then deploy the sample service to view how GLBC allows access to services in a multi-cluster ingress scenario.

<br>

**Tutorial Contents:**
* [Prerequisites](#prerequisites)
* [Install](#installation)
* [Provide GLBC credentials](#provide-glbc-with-aws-credentials-and-configuration)
* [Run kcp and GLBC](#run-kcp-and-glbc)
* [Deploy the sample service](#deploy-the-sample-service)
* [Verify sample service deployment](#verify-sample-service-deployment)
* [Demo: **Providing ingress in a multi-cluster ingress scenario**](#main-use-case)

---

## Prerequisites
- Install [kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl).
- Install Go 1.18 or higher. This is the version used in kcp-glbc as indicated in the [`go.mod`](/go.mod) file.
- Install the [yq](https://github.com/mikefarah/yq) command-line YAML processor.
- Have an AWS account, a DNS Zone, and a subdomain of the domain being used. You will need this in order to instruct GLBC to make use of your AWS credentials and configuration.
- Add the `kcp-glbc/bin` directory to your `$PATH`

<br>

## Installation

### Provide GLBC with AWS credentials and configuration

Before deploying `kcp` and GLBC, we will want to provide GLBC with AWS credentials and configuration. The easiest way to do this is to perform the following steps:

1. Open the `kcp-glbc` project in your IDE.
1. Navigate to the `./config/deploy/local/kcp-glbc/aws-credentials.env` environment file.
1. Enter your `AWS access key ID` and `AWS Secret Access Key` as indicated in the example below:

     ```bash
      AWS_ACCESS_KEY_ID=EXAMPLEID2DJ3rSA3E
      AWS_SECRET_ACCESS_KEY=EXAMPLEKEYIEI034+fETFDS34QFAD0IAO
      ```
1. Navigate to `./config/deploy/local/kcp-glbc/controller-config.env` and change the fields to resemble something similar to following:

   ```bash
   AWS_DNS_PUBLIC_ZONE_ID=Z0668753LU5S8CXOSZR0
   GLBC_DNS_PROVIDER=aws
   GLBC_DOMAIN=cz.hcpapps.net
   GLBC_EXPORT=glbc-root-kuadrant
   GLBC_LOGICAL_CLUSTER_TARGET=*
   GLBC_TLS_PROVIDER=glbc-ca
   GLBC_WORKSPACE=root:kuadrant
   HCG_LE_EMAIL=kuadrant-dev@redhat.com
   NAMESPACE=kcp-glbc
   ```

   The fields that might need to be edited include:
   - Replace `<AWS_DNS_PUBLIC_ZONE_ID>` with your own hosted zone ID
   - Replace `<GLBC_DOMAIN>` with your specified subdomain

<br>

### Run `kcp` and GLBC
Clone the repo. You would need to set `NUM_CLUSTERS=2`, and for simplicity set `RUN_GLBC=true` which will immediately run GLBC after
the local-setup script has completed. After running the following command, we will have `kcp` and GLBC running:

```bash
make local-setup NUM_CLUSTERS=2 DEPLOY_GLBC=true
```
> NOTE: If errors are encountered during the local-setup, refer to the [Troubleshooting Installation](/docs/troubleshooting.md) document.

This script performs the following actions: 
* Builds all the binaries
* Deploys two Kubernetes 1.22 clusters locally using [kind](https://kind.sigs.k8s.io/)
* Deploys and configures the ingress controllers in each cluster
* Downloads kcp at the latest version integrated with GLBC
* Starts the kcp server
* Adds Kind clusters as sync targets 
* Deploy GLBC dependencies (`cert-manager`) into the `kcp-glbc` workspace.

* Additionally, we would have deployed GLBC.

<br>

### Deploy the sample service

Now we will attempt to deploy the sample service. You can do this by running the following command in a new tab in the terminal:

   ```bash
   ./samples/location-api/sample.sh
   ```
After the sample service has been deployed, we are presented with the following output of what was done, and some useful commands:

```bash
before running this script, ensure that you have set the flag --advanced-scheduling=true when starting GLBC
Press enter to continue
Current workspace is "root:kuadrant" (type "root:universal").
creating locations for sync targets in root:kuadrant workspace
location.scheduling.kcp.dev/kcp-location-1 created
location.scheduling.kcp.dev/kcp-location-2 created
creating placement in home workspace
Current workspace is "root:users:zu:yc:kcp-admin".
creating apibindings in home workspace
apibinding.apis.kcp.dev/kubernetes created
apibinding.apis.kcp.dev/glbc created
placement.scheduling.kcp.dev/placement-1 created
placement.scheduling.kcp.dev "default" deleted
deploying workload resources in home workspace
service/echo created
deployment.apps/echo created
ingress.networking.k8s.io/echo created

=== useful commands:
  - watch -n1 "curl -k https://ccdncilkjgm3i9jmt6c0.cz.hcpapps.net" (N.B. Don't start this before A record exists in route53)
  - watch -n1 'kubectl get dnsrecords echo -o yaml | yq eval ".spec" -'
  - watch -n1 'dig ccdncilkjgm3i9jmt6c0.cz.hcpapps.net'
  - watch -n1 -d 'kubectl get ingress echo -o yaml | yq eval ".metadata" - | grep -v "kubernetes.io"'


Press enter to trigger migration from kcp-cluster-1 to kcp-cluster-2
```


The sample script will remain paused until we press the enter key to migrate the workload from one cluster to the other. However, we will not perform this action just yet.

<br>

### Verify sample service deployment

1. In a new terminal, verify that the ingress was created after deploying the sample service:
   ```bash
   export KUBECONFIG=.kcp/admin.kubeconfig                                         
   ./bin/kubectl-kcp workspace use '~'
   kubectl get ingress
   ```
   ```bash
   NAME       AGE
   echo       81s
   ```

1. Verify that the DNS record was created:
   ```bash
   export KUBECONFIG=.kcp/admin.kubeconfig                                         
   ./bin/kubectl-kcp workspace use '~'
   kubectl get dnsrecords echo -o yaml
   ```
   We might not get an output just yet until the DNS record exists in `route53`. This may take several minutes.

1. Alternatively, you can also view if the DNS record was created, by  in your AWS domain .

   ![Screenshot from 2022-08-02 12-26-19](https://user-images.githubusercontent.com/73656840/182363808-558f8a40-4ed6-4e08-9c02-1d74b6209b46.png)

------

## Main Use Case

### Demo: Providing ingress in a multi-cluster ingress scenario

This section will show how GLBC is used to provide ingress in a multi-cluster ingress scenario.

For this tutorial, after following along the with the [Installation](#installation) section of this document, we should already have `kcp` and GLBC running, and also have had deployed the sample service which would have created a placement resource, an ingress named *"echo"* and a DNS record. Note, the "default" namespace is where we are putting all the sample resources at the moment.

<br>

#### Viewing the "default" namespace

We will run the following commands in a new tab:

   ```bash
   export KUBECONFIG=.kcp/admin.kubeconfig                                         
   ./bin/kubectl-kcp workspace use '~'
   kubectl get ns default -o yaml
   ```
As we can see, there is a label named something like: `*state.workload.kcp.dev/5MivhNIs7DjM7dK95I2K7TpWe7aUGMU4WHqjWn: Sync*`:

GLBC is telling `kcp` where to sync all of the work resources in the namespace. Meaning, since the `state` label in the namespace is set to `5MivhNIs7DjM7dK95I2K7TpWe7aUGMU4WHqjWn` (kcp-cluster-1) , the `state` label in the ingress will also have `5MivhNIs7DjM7dK95I2K7TpWe7aUGMU4WHqjWn` (kcp-cluster-1) set to it. 

<br>

#### Watching the ingress and the DNS record

We can run the watch command in a new tab to start watching the ingress:

   ```bash
   export KUBECONFIG=.kcp/admin.kubeconfig                                         
   ./bin/kubectl-kcp workspace use '~'
   watch -n1 -d 'kubectl get ingress echo -o yaml | yq eval ".metadata" - | grep -v "kubernetes.io"'
   ```

As we can see in the first annotation, the load balancer for `kcp-cluster-1` will have an IP address, after the DNS record is created:

   ![Screenshot from 2022-09-09 19-11-06](https://user-images.githubusercontent.com/73656840/189416748-fb94527e-509c-44c0-b680-300691721acb.png)


Alternatively, we can also run the following command in another tab to start watching the DNS record in real-time:

   ```bash
   export KUBECONFIG=.kcp/admin.kubeconfig                                         
   ./bin/kubectl-kcp workspace use '~'
   watch -n1 'kubectl get dnsrecords echo -o yaml | yq eval ".spec" -'
   ```
   
<br>

#### Curl the running domain

Now that the DNS record has been successfully created, in a new tab in the terminal, we can curl the domain to view it. To do this, we will run the following watch command that is outputted to our terminal and will look similar to the following:

   ```bash
   watch -n1 "curl -k https://cbkgg75kjgmah1mbpvsg.cz.hcpapps.net"
   ```

   This will curl the domain which will give an output similar to the following:

   ![Screenshot from 2022-08-02 12-44-15](https://user-images.githubusercontent.com/73656840/182368772-8a08a197-66d9-4d9c-9747-74ddaad0e4d7.png)

This means that kcp-cluster-1 is up and running correctly.

<br>

#### Migrating workload from one cluster to another cluster

As we continue with the following steps, we will want to be observing the tab where we are watching our domain. This way, we will notice that during the workload migration, there is no interruptions and no downtime.

To proceed with the workload migration, we will go to the tab where we deployed the sample service, and press the enter key to "trigger migration from `kcp-cluster-1` to `kcp-cluster-2`. This deletes `placement-1` and creates `placement-2` which points to `kcp-cluster-2`. This will also change the label in the "default" namespace mentioned before: `*state.internal.workload.kcp.dev/5MivhNIs7DjM7dK95I2K7TpWe7aUGMU4WHqjWn: Sync*` to the other cluster.

   ![Screenshot from 2022-09-09 19-26-36](https://user-images.githubusercontent.com/73656840/189419200-17dd3086-82f1-4bf7-ae95-70bc4f3e8b87.png)


In the tab where we are watching the ingress, we can observe that the label in the ingress has changed from `kcp-cluster-1` to `kcp-cluster-2`. KCP has propagated that label down from the namespace to everything in it. Everything in the namespace gets the same label. Because of that label, `kcp` is syncing it to `kcp-cluster-2`.

   ![Screenshot from 2022-09-09 19-22-43](https://user-images.githubusercontent.com/73656840/189418521-66a70b94-4f7e-47de-bbd0-1987d0a2dcce.png)


Moreover, in the annotations we also have a status there for `kcp-cluster-2` and it has an IP address in it meaning that it has indeed synced to `kcp-cluster-2`. We will also find another annotation named "*deletion.internal.workload.kcp.dev/kcp-cluster-1*", which is code coming from the GLBC which is saying "Don't remove this work from `kcp-cluster-1` until the DNS has propagated."

For that reason we can also observe another annotation named "*finalizers.workload.kcp.dev/kcp-cluster-1: kuadrant.dev/glbc-migration*" which remains there because GLBC is saying to `kcp` "Don't get rid of this yet, as we're waiting for it to come up to another cluster before you remove it from `kcp-cluster-1`. After it has completely migrated and sufficient time has been allowed for DNS propagation, the finalizer for `kcp-cluster-1` will no longer be there and the workload will be deleted from `kcp-cluster-1`. Account for the DNS propagation time of the TTL (usually 60 seconds) * 2.

   ![Screenshot from 2022-09-09 19-25-07](https://user-images.githubusercontent.com/73656840/189418887-2efd4977-17dc-4d1e-b888-ca32b58d862a.png)


We will notice that in the tab where we are curling the domain, we will always be getting a response back because the graceful migration is active. Meaning, even after the workload has been migrated, and the DNS record is updated, we will keep receiving a response without any interruption even after `kcp-cluster-1` has been deleted. This can be observed in our curl:

   ![Screenshot from 2022-08-02 12-55-24](https://user-images.githubusercontent.com/73656840/182368597-1ec0ade2-9849-4414-849f-ac342680d11b.png)

This shows that the workload has successfully migrated from `kcp-cluster-1` to `kcp-cluster-2` without any interruption.

<br>

#### Clean-up

After finishing with this tutorial, we can go back to our tab where we deployed the sample service, and press the enter key to reset and undo everything that was done from running the sample.

   ![Screenshot from 2022-09-09 19-29-44](https://user-images.githubusercontent.com/73656840/189419668-cb3d8d67-34ef-4148-8df1-f54fdb4bf196.png)

## Other setup options with CRC

### Using CRC sync target (optional)

The default local setup uses a single sync target backed by a [kind](https://kind.sigs.k8s.io/) physical cluster, you can alternatively use a [crc](https://crc.dev/crc/) cluster by runnning:

```bash
make local-setup USE_CRC=true
```

### Add CRC sync target to existing local setup (optional)

With a running local setup i.e. you have successfully executed `make local-setup`, you can run the following to create and add an additional sync target backed by a [crc](https://crc.dev/crc/) cluster:

```bash
./utils/local-setup-add-crc-cluster.sh
```

```bash
$ kubectl get synctargets -o wide
NAME              LOCATION          READY   SYNCED API RESOURCES
kcp-cluster-1     kcp-cluster-1     True    ["deployments.apps","ingresses.networking.k8s.io","secrets","services"]
kcp-cluster-crc   kcp-cluster-crc   True    ["deployments.apps","ingresses.networking.k8s.io","secrets","services"]
```

You must have crc installed(https://crc.dev/crc/), and have your openshift pull secret(https://cloud.redhat.com/openshift/create/local) stored locally in `~/pull-secret`. 
Please check the script comments for any version requirements.