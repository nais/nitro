# Nitro

Application for provisioning Kubernetes clusters and configuration using  [Flatcar](https://www.flatcar.org/) nodes.
Use in conjunction with a repo containing cluster definitions. The nodes are configured using [Ignition](https://www.flatcar.org/docs/latest/provisioning/ignition/).

### Create new cluster
1. Create a repo containing cluster definitions and variables, including node names for apiserver, etcd, workers and so on. See [examples](./examples)
2. Add deployer ssh key to a local directory (default ./id_deployer_rsa)
3. Add the new cluster to your kubeconfig file
```
kubectl config set-cluster <cluster name> --insecure-skip-tls-verify=true --server=https://127.0.0.1:6443
kubectl config set-context <cluster name> --user=nais-user --cluster=<cluster name> --namespace=kube-system
```
6. Create definition and var files for new cluster in `./vars` and `./clusters` vars.yaml). See [examples][./examples]

7. Create workflow for provisioning. See [examples](./examples/workflow.yaml)

### Add worker node to existing cluster
1. Create a new node

2. When the VM is up, add it to [cluster name]-nodes.yaml and push it to github.
This will trigger the pipeline. The workflow will generate a config for the new
node, push it to the VM and trigger a provision.

### Add etcd node to existing cluster

As there is no support for this in nitro, we need to do some manual patching
along the way.

1. Create the new node and add it to the cluster file

2. Add member to etcd cluster from one of the existing etcd-nodes:
```
etcdctl member add <nodename> --peer-urls https://<nodename>:2380
```

3. Delete all certificates from existing etcd-nodes as they are missing the new
   node

4. Run the nitro workflow

5. When the workflow is done, log in to the new etcd node and delete
   /var/lib/etcd/member and set the /etc/systemd/system/etcd.service
   initial-cluster-state to existing restart the etcd service

### Move api-server

These steps also require some amount of manual lay on hands.

In order to move apiserver you will need to:

1. Create a new node in the cluster description with ´location: azure´ and with a
   node name of your choosing, e.g apiserver-1

2. copy all certificates from the old apiserver except the api server certificate and the
   apiserver kubelet certificate over to the new apiserver instance.

3. Delete the kubelet certificates on the worker nodes

4. Run the nitro workflow, which will now reprovision the missing certificates
