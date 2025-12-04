#!/usr/bin/env bash

helm upgrade --install mongo bitnami/mongodb-sharded -f mongo-values.yaml --namespace mongo-system --create-namespace --version 6.0.0
helm upgrade --install redis bitnami/redis -f redis-values.yaml --namespace redis-system --create-namespace  --version 17.9.4
helm upgrade --install es bitnami/elasticsearch -f elasticsearch.yaml  --namespace es-system --create-namespace --version 19.9.5
helm upgrade --install es bitnami/elasticsearch -f elasticsearch.yaml  --namespace es-system --create-namespace --version 19.9.5

helm upgrade --install kibana bitnami/kibana -f kibana-values.yaml --namespace es-system --create-namespace --version 10.4.1
