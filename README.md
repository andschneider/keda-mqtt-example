# KEDA MQTT example

An example showing how to use KEDA to scale a Deployment based on a MQTT retained message.

*!! The scaler is currently being developed and can be found on my branch [here](https://github.com/andschneider/keda/tree/mqtt-scaler). You'll need to build and deploy KEDA manually for testing this !!*

## Components

- `Mosquitto MQTT broker`: The [mosquitto](https://mosquitto.org/) broker is deployed into the cluster and exposed as a service for the other components to use.
- `subscriber`: A deployment that subcribes to a topic and receives messages from the broker. This is target for KEDA to scale.
- `scaler`: The KEDA ScaledObject, which has the required configuration for the broker, topic, and desired replicas to scale to.
- `publish-job`: A Kubernetes job which will publish 100 MQTT messages.
- `publish-retain`: A Kubernetes job which will publish a single retained message (tells KEDA to scale *up*).
- `publish-clear`: A Kubernetes job which will clear a retained message (tells KEDA to scale *down*).

## Setup

This setup will go through deploying a MQTT broker in the cluster and deploying a MQTT subscriber and publisher.

### namespace

Create a new namespace in the cluster: `kubectl create namespace mqtt`

> Note: This isn't completely necessary. If you aren't able to create a namespace or have to use a specific one, just change all the `metadata.namespace` fields in the .yaml files.

### broker

Deploy the Mosquitto broker and expose as a Service with: 

```bash
kubectl apply -f k8s/mosquitto.yaml
```

The Deployment is *not* configured with persistent storage. This means that if the pod running the broker is ever restarted it will lose any retained messages that it was currently storing. Effectively this will inform KEDA to scale the subscriber to zero, and while useful for debugging, it is likely not what you'd want in production. Please see the [mosquitto configuration](https://mosquitto.org/man/mosquitto-conf-5.html) for how to enable persistence and [Kubernetes documentation](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#types-of-persistent-volumes) about the available options for persistent storage.

### configuration

Each of the following components interact with the deployed broker by specifying a `topic`. The default topic is `keda-mqtt-example` but feel free to change it. For the scaler it is specified in its `metadata` field, for the others it is specified in the `containers.args` field.

### subscriber

Deploy the MQTT subscriber, which will simply listen to a topic and print any messages it receives, with:

```bash
kubectl apply -f k8s/mqtt-subscriber.yaml
```

### scaler

Deploy the KEDA scaler with:

```bash
kubectl apply -f k8s/mqtt-scaler.yaml
```

This will then scale the subscriber to zero and begin checking the topic for a retained message.

### publish retained

Run a Kubernetes job that sends a *retained* MQTT message:

```bash
kubectl apply -f k8s/mqtt-publish-job-retained.yaml
```

The next time the scaler runs it will see that there is a retained message and scale the subscriber to the desired number of replicas.

### publish

Run a Kubernetes job that sends 100 MQTT messages (one every second):

```bash
kubectl apply -f k8s/mqtt-publish-job.yaml
```

If you want to run it again, delete the old job first:

```bash
kubectl delete -f k8s/mqtt-publish-job.yaml
```

### clear retained

Run a Kubernetes job that will *remove* any retained message currently saved:

```bash
kubectl apply -f k8s/mqtt-clear-retained.yaml
``` 

## CLI

The publish and subscribe Kubernetes resources are simply a CLI bundled in a Docker image. It might be preferable to run the CLI locally for development, and in combination with port-forwarding, generally will be quicker and easier to use to see what's going on.
 
### build

The build is configured in the `Makefile` and it's currently targeted at linux. To build, run:

```bash
make build
```

You should now have a `keda-mqtt` binary in the `bin` directory.

### usage

- To *subscribe* to a topic run `./bin/keda-mqtt -host localhost:1883 -topic keda-mqtt-example`
- To *publish* to a topic run `./bin/keda-mqtt -host localhost:1883 -topic keda-mqtt-example -pub [-count int]`
- To *publish* a *retained* message to a topic run `./bin/keda-mqtt -host localhost:1883 -topic keda-mqtt-example -pub -retain`
- To *clear* a *retained* message in a topic run `./bin/keda-mqtt -host localhost:1883 -topic keda-mqtt-example -clear`

> Note: All of the examples above are assuming a broker is available at `localhost:1883`. To port-forward to the hosted broker you can run `kubectl port-forward svc/mosquitto-service 1883:1883 -n mqtt`

> Note 2: By default, the broker the CLI will attempt to use is the one provided by Mosquitto at `test.mosquitto.org`. If you ommit the `-host` flag it will connect over HTTP to it.

## Cleanup

To remove the created resources, run:

```bash
kubectl delete -f k8s
kubectl delete namespace mqtt
```
