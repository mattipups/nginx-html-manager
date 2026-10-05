.PHONY: init lock build up down test scan smoke logs
init:
	./scripts/init.sh
lock:
	./scripts/lock-images.sh
build:
	./scripts/compose.sh build --pull
up:
	./scripts/compose.sh up -d --build --wait
down:
	./scripts/compose.sh down
test:
	./scripts/test.sh
scan:
	./scripts/scan.sh
smoke:
	./scripts/smoke.sh
logs:
	./scripts/compose.sh logs -f
