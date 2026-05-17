# 注意: 此 demo 推荐在 Linux 跑 (Mac 上 libSystem malloc 不走全局符号, 拦不住).
# Mac 用户请用 ./run-in-docker.sh 跑.

.PHONY: linux run-linux docker clean

linux: malloc-limit.so victim

malloc-limit.so: malloc-limit.c
	gcc -shared -fPIC malloc-limit.c -o malloc-limit.so -ldl

victim: victim.c
	gcc victim.c -o victim

run-linux: linux
	@echo "--- 跑 hook 拦截 (上限 5MB) ---"
	@MEM_LIMIT_MB=5 LD_PRELOAD=./malloc-limit.so ./victim || true

docker:
	./run-in-docker.sh

clean:
	rm -f malloc-limit.so malloc-limit.dylib victim
