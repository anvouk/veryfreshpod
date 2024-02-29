# veryfreshpod

veryfreshpod aims at simplifying Kubernetes development by automatically deploying new built images for already deployed
containers.

Example workflow:
- your cluster is running a DeploymentSet with `myimage:test`
- you see a bug, fix it, and build a new image tagged `myimage:fixed`
- veryfreshpod automatically detects the change and replaces all references for `myimage:test` to `myimage:fixed` for
all StatefulSets and Deployments present in your dev cluster.
- Kubernetes then begins replacing pods as it would with a regular deploy command.
- repeat!

Basically, after the first deploy, you do not ever need to perform another deploy again. This also works by pulling a
new image with docker.

## Install veryfreshpod

> **Important**: ensure your cluster is using docker instead of containerd. see k3s docker setup below.

```bash
kubectl apply -f https://raw.githubusercontent.com/anvouk/veryfreshpod/master/manifest/deployment.yaml
```

## Remove veryfreshpod

```bash
kubectl remove -f https://raw.githubusercontent.com/anvouk/veryfreshpod/master/manifest/deployment.yaml
```

## History

veryfreshpod is inspired by, the now defunct, [GCP freshpod](https://github.com/GoogleCloudPlatform/freshpod) and it
improves upon its features.

freshpod acted on pods directly, however veryfreshpod acts on statefulsets and deployments and leaves the restart and
update of pods to the kubernetes scheduler.

The way is works is by mapping all statefulsets and deployments and which images the use. veryfreshpod then watches
docker for new pulled or tagged images and if it finds a new tag for an already deployed image in the cluster, it updates
the corresponding statefulset/deployment to use this new image.

I developed veryfreshpod to quicken development on a k3s cluster backed with docker, as such, I only really tested it
with this configuration. It should still work without problems on other distributions (minikube, etc) as long as
docker is being used instead of containerd.

## k3s docker setup

Check if k3s is already using docker. If running `docker ps` shows something like this:
```bash
user@ubuntu:~$ docker ps
CONTAINER ID   IMAGE                        COMMAND                  CREATED          STATUS          PORTS     NAMES
54a0ca1683f5   d1e26b5f8193                 "/entrypoint.sh --gl…"   30 seconds ago   Up 29 seconds             k8s_traefik_traefik-57c84cf78d-lk4rz_kube-system_e49427cf-bdf2-40d1-99a9-bcdf6e0186e9_0
a74cb872c364   rancher/mirrored-pause:3.6   "/pause"                 30 seconds ago   Up 30 seconds             k8s_POD_traefik-57c84cf78d-lk4rz_kube-system_e49427cf-bdf2-40d1-99a9-bcdf6e0186e9_0
d2539c0d7649   af74bd845c4a                 "entry"                  31 seconds ago   Up 30 seconds             k8s_lb-tcp-443_svclb-traefik-3c962c7c-zhq68_kube-system_2e8dd543-0067-445f-95f7-0d7a6617a55c_0
45d7e0b712a0   af74bd845c4a                 "entry"                  31 seconds ago   Up 30 seconds             k8s_lb-tcp-80_svclb-traefik-3c962c7c-zhq68_kube-system_2e8dd543-0067-445f-95f7-0d7a6617a55c_0
f228a9137b1d   rancher/mirrored-pause:3.6   "/pause"                 31 seconds ago   Up 30 seconds             k8s_POD_svclb-traefik-3c962c7c-zhq68_kube-system_2e8dd543-0067-445f-95f7-0d7a6617a55c_0
122aea6493d0   817bbe3f2e51                 "/metrics-server --c…"   34 seconds ago   Up 33 seconds             k8s_metrics-server_metrics-server-68cf49699b-2fz8x_kube-system_ae804cee-8166-449b-9600-85e30da5bb53_0
7ce171da971e   rancher/mirrored-pause:3.6   "/pause"                 34 seconds ago   Up 34 seconds             k8s_POD_metrics-server-68cf49699b-2fz8x_kube-system_ae804cee-8166-449b-9600-85e30da5bb53_0
6224284c29ec   b29384aeb4b1                 "local-path-provisio…"   35 seconds ago   Up 35 seconds             k8s_local-path-provisioner_local-path-provisioner-76d776f6f9-cnqfv_kube-system_1f5b9603-1d4c-4ab3-a9f2-cfea3ec303ac_0
932ad7039069   rancher/mirrored-pause:3.6   "/pause"                 36 seconds ago   Up 35 seconds             k8s_POD_local-path-provisioner-76d776f6f9-cnqfv_kube-system_1f5b9603-1d4c-4ab3-a9f2-cfea3ec303ac_0
10e91d576fdf   ead0a4a53df8                 "/coredns -conf /etc…"   37 seconds ago   Up 36 seconds             k8s_coredns_coredns-59b4f5bbd5-qjtzs_kube-system_5cca5e66-674d-4d81-b0c9-ec803d919248_0
35da8945fe8a   rancher/mirrored-pause:3.6   "/pause"                 37 seconds ago   Up 36 seconds             k8s_POD_coredns-59b4f5bbd5-qjtzs_kube-system_5cca5e66-674d-4d81-b0c9-ec803d919248_0
```
it means k3s you're already set and veryfreshpod should work once installed.

If you don't see kubernetes containers, you need to ensure k3s is starting with the `--docker` flag and docker is installed
on the host.

The easiest way is to re-run the k3s install script and telling it to use docker:
```bash
export INSTALL_K3S_EXEC="--docker"
./k3s-installer.sh
```

Alternatively, you can directly edit the generated systemd unit file at `/etc/systemd/system/k3s.service` and add the
`--docker` flag there.

## Future work

- Add DaemonSets support: at the moment, only Deployments and Statefulsets are supported.
- Add containerd support instead of docker

## Non-goals

- Security: veryfreshpod is targeted to personal dev boxes, it is not meant for any prod usages, ever.
- Multi-node support: only clusters running on a single node will be supported.

## License

Apache Version 2.0
