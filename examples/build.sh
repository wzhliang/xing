#!/bin/bash
REPO=hub.c.163.com/borlandc

for s in server server1 evt-server; do
	GOOS=linux GOARCH=amd64 go build $s.go
	cat Dockerfile.server | APP=$s mo > Dockerfile
	docker build -t $REPO/xing-$s:latest .
	docker push $REPO/xing-$s:latest
done

for c in client client1 notify noreply; do
	GOOS=linux GOARCH=amd64 go build $c.go
	cat Dockerfile.client | APP=$c N=99999 mo > Dockerfile
	docker build -t $REPO/xing-$c:latest .
	docker push $REPO/xing-$c:latest
done

rm Dockerfile
