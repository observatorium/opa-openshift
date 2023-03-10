#!/bin/bash

# Runs a semi-realistic integration test with a producer generating logs
# all being authenticated via Dex and authorized with opa-openshift.

set -euo pipefail

# shellcheck disable=SC1091
source .bingo/variables.env

result=1
trap 'kill $(jobs -p); cleanup; exit $result' EXIT

# shellcheck disable=SC2317
cleanup(){
    echo "-------------------------------------------"
    echo "- Cleanup users, role and rolebindings... -"
    echo "-------------------------------------------"
    kubectl delete -f ./test/config/openshift.yaml
}

echo "-------------------------------------------"
echo "- Prepare users, role and rolebindings... -"
echo "-------------------------------------------"
kubectl apply -f ./test/config/openshift.yaml

($DEX serve ./test/config/dex.yaml > "$LOG_DIR"/dex.log 2>&1) &

echo "-------------------------------------------"
echo "- Waiting for Dex to come up...           -"
echo "-------------------------------------------"

until curl --output /dev/null --silent --fail --insecure https://127.0.0.1:5556/dex/.well-known/openid-configuration; do
  printf '.'
  sleep 1
done

echo "-------------------------------------------"
echo "- Getting authentication token...         -"
echo "-------------------------------------------"
sleep 2

token=$(curl --request POST \
  --silent \
  --cacert ./tmp/certs/ca.pem \
  --url https://127.0.0.1:5556/dex/token \
  --header 'content-type: application/x-www-form-urlencoded' \
  --data grant_type=password \
  --data username=admin@example.com \
  --data password=password \
  --data client_id=test \
  --data client_secret=ZXhhbXBsZS1hcHAtc2VjcmV0 \
  --data scope="openid profile groups email" | sed 's/^{.*"id_token":[^"]*"\([^"]*\)".*}/\1/')

(
  api \
    --web.listen=0.0.0.0:8443 \
    --web.internal.listen=0.0.0.0:8448 \
    --web.healthchecks.url=http://127.0.0.1:8443 \
    --tls.server.cert-file=./tmp/certs/server.pem \
    --tls.server.key-file=./tmp/certs/server.key \
    --tls.healthchecks.server-ca-file=./tmp/certs/ca.pem \
    --logs.read.endpoint=http://127.0.0.1:3100 \
    --logs.tail.endpoint=http://127.0.0.1:3100 \
    --logs.write.endpoint=http://127.0.0.1:3100 \
    --rbac.config=./test/config/rbac.yaml \
    --tenants.config=./test/config/tenants.yaml \
    --log.level=debug > "$LOG_DIR"/observatorium.log 2>&1
) &


(
  $LOKI \
    -log.level=info \
    -target=all \
    -config.file=./test/config/loki.yml > "$LOG_DIR"/loki.log 2>&1
) &

(
  ./opa-openshift \
      --openshift.kubeconfig="$HOME"/.kube/config \
      --openshift.mappings=application="observatorium.openshift.io"  \
      --openshift.mappings=infrastructure="observatorium.openshift.io" \
      --openshift.mappings=audit="observatorium.openshift.io" \
      --opa.package=observatorium \
      --log.level=debug \
      --debug.token="$(oc create token the-cluster-admin -n default)" \
      --web.listen=:8080 > "$LOG_DIR"/opa-openshift.log 2>&1
) &

echo "-------------------------------------------"
echo "- Waiting for dependencies to come up...  -"
echo "-------------------------------------------"
sleep 10

until curl --output /dev/null --silent --fail http://127.0.0.1:8081/ready; do
  printf '.'
  sleep 1
done

echo "-------------------------------------------"
echo "- Application Logs tests                  -"
echo "-------------------------------------------"

if $UP \
  --listen=0.0.0.0:8888 \
  --endpoint-type=logs \
  --tls-ca-file=./tmp/certs/ca.pem \
  --endpoint-read=https://127.0.0.1:8443/api/logs/v1/application/loki/api/v1/query \
  --endpoint-write=https://127.0.0.1:8443/api/logs/v1/application/loki/api/v1/push \
  --period=500ms \
  --initial-query-delay=250ms \
  --latency=10s \
  --duration=10s \
  --log.level=debug \
  --name=up_test \
  --labels='foo="bar"' \
  --logs="[\"$(date '+%s%N')\",\"log line 1\"]" \
  --token="$token"; then
  result=0
  echo -e "\n"
  echo "-------------------------------------------"
  echo "- tests: OK                               -"
  echo "-------------------------------------------"
else
  result=1
  echo -e "\n"
  echo "-------------------------------------------"
  echo "- tests: FAILED                           -"
  echo "-------------------------------------------"
  exit 1
fi

echo -e "\n"
echo "-------------------------------------------"
echo "- All tests: OK                           -"
echo "-------------------------------------------"
exit 0
