# Sinozo Java application

Reusable application Chart for independently deployed Java services. The
application image determines the JDK. Current maintained JDK 8 (8u191+), 11,
17 and 21 can use the default container/percentage JVM options; change the
options to match the actual image and JVM distribution.

- Default resources: Deployment and Service, port 28080, one replica.
- `service.port` also sets SERVER_PORT and the default probe, monitoring and
  Ingress backend port. Application-specific port configuration must agree.
- `application.name` defaults to the Release name. `application.profile`,
  Nacos namespace/address, resources and routing are environment-specific.
- Java uses JAVA_TOOL_OPTIONS by default, read directly by the JVM. Do not
  also add the same flags in JAVA_OPTS or command arguments.
- Nacos, authentication, SkyWalking, heap dump, probes, monitoring and Ingress
  are opt-in. Authentication only references an existing namespaced Secret;
  no username/password environment entries are emitted when disabled.
- SkyWalking adds an agent init container, an emptyDir mount and a javaagent
  option. Configure a reachable OAP endpoint and enough application memory.
- Ingress items support ALB and Nginx annotations, multiple hosts and paths.
  Backends always reference this Chart's Service. Disabled TLS emits no tls
  stanza. ALB may use a cloud certificate without secretName; Nginx requires
  an existing TLS Secret. Certificate/controller setup is external.
- Native env/valueFrom/envFrom, command/args, volumes/mounts, scheduling,
  security contexts, init containers and sidecars remain editable.
- Resources are stable between identical renders; no timestamp annotations
  force a rollout on every preview. Never place passwords or tokens in JVM
  flags or ordinary values.

See docs/02-cicd/JAVA_SKYWALKING_TEMPLATE_2026-10-17.md in AxiOps for complete
test/production values and ALB/Nginx examples. These examples live outside
this Chart so they are not accidentally used as deployment defaults.
