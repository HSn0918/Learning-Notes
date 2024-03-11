```shell
docker network create elastic
```
```yaml
version: '3'
services:
  elasticsearch:
    image: docker.elastic.co/elasticsearch/elasticsearch:8.6.0
    container_name: elasticsearch
    networks:
      - elastic
    ports:
      - 9200:9200
    environment:
      - discovery.type=single-node
      - ES_JAVA_OPTS=-Xms1g -Xmx1g
      - xpack.security.enabled=false
    volumes:
      - ./plugins:/usr/share/elasticsearch/plugins
    stdin_open: true
    tty: true
networks:
  elastic:
    external: true
```
```bash
docker run  --name kibana  --net elastic  -p 5601:5601 docker.elastic.co/kibana/kibana:8.6.0
```