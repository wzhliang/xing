#!/bin/bash
REPO=hub.c.163.com/borlandc/xing
SERVERS="server server1 evt-server strm-consumer"
CLIENTS="client client1 notify noreply strm-producer multi-client"

for s in $SERVERS; do
	GOOS=linux GOARCH=amd64 go build $s.go
done
for c in $CLIENTS; do
	GOOS=linux GOARCH=amd64 go build $c.go
done

for s in $SERVERS; do
	cat Dockerfile.server | APP=$s mo > Dockerfile
	docker build -t $REPO/xing-$s:latest .
	docker push $REPO/xing-$s:latest
done

for c in $CLIENTS; do
	cat Dockerfile.client | APP=$c N=99999 mo > Dockerfile
	docker build -t $REPO/xing-$c:latest .
	docker push $REPO/xing-$c:latest
done

rm Dockerfile
