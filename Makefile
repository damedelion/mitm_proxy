gen:
	sh gen_ca.sh

clean:
	rm ca.crt ca.key cert.key

docker-build:
	-docker rm mitm_proxy
	-docker rmi damedelion/mitm_proxy
	docker build -t damedelion/mitm_proxy .

docker-run:
	docker run -d --name mitm_proxy -p 8080:8080 damedelion/mitm_proxy

docker-stop:
	docker stop mitm_proxy

docker-start:
	docker start mitm_proxy

docker-logs:
	docker logs mitm_proxy