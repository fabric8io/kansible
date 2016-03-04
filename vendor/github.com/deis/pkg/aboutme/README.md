# aboutme: Self-discovery for Kubernetes containers

This library provides hooks for containers to learn about themselves and
their pods.

Essentially, this uses environmental data to connect to a k8s API server
and find out about the current pod's configuration.
