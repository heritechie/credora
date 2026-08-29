# Deploy

Local and production deployment configuration for Credora.

Local infrastructure is not configured yet. Once the engine implementation
phase begins, this directory will contain:

- Dockerfile(s) for `services/engine`
- Docker Compose for local PostgreSQL + engine
- any supporting deployment scripts

No Kubernetes, Redis, Kafka, RabbitMQ, or Temporal will be added unless a
concrete requirement emerges.