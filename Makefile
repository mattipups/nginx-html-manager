.PHONY: init lock build up down test scan smoke logs chart
init:
	bash scripts/init.sh
lock:
	bash scripts/lock-images.sh
build:
	bash scripts/compose.sh build --pull
up:
	bash scripts/compose.sh up -d --build --wait
down:
	bash scripts/compose.sh down
test:
	bash scripts/test.sh
scan:
	bash scripts/scan.sh
smoke:
	bash scripts/smoke.sh
logs:
	bash scripts/compose.sh logs -f
chart:
	bash scripts/check-chart.sh
