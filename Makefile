# This is the BUILD target, do not remove it, and do not modify it's name
go-server-build:
	GOARCH=`dpkg --print-architecture` TMPDIR=${TMP} go build
.PHONY: go-server-build

# This is the RUN target, do not remove it, and do not modify it's name
go-server-run:
	@GOARCH=`dpkg --print-architecture` TMPDIR=${TMP} go run server.go 2>&1 | tee -a $$LOG_TO &
.PHONY: go-server-run

# This is the STOP target, do not remove it, and do not modify it's name
go-server-stop:
	@-pkill -f ".tmp/go-build.*/exe/server" > /dev/null 2>&1 || true
.PHONY: go-server-stop

# This is the RESTART target, do not remove it, and do not modify it's name
go-server-restart:
	@echo "Restarting Go server..."
	@$(MAKE) --no-print-directory -s go-server-stop
	@sleep 0.5
	@$(MAKE) --no-print-directory -s go-server-run
	@echo "Go server restarted."
.PHONY: go-server-restart
