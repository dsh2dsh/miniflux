.PHONY: e2e kill-e2e clean-e2e clean-e2e-error
e2e:
	dropdb --if-exists -U postgres miniflux_test
	createdb -U postgres -O miniflux -E UTF-8 --locale en_US.UTF-8 \
		-T template0 miniflux_test
	go run ./cmd/api -local
	go test -v -count=1 -tags e2e ${E2E_TEST_ARGS} ./internal/api || \
		${MAKE} clean-e2e-error
	${MAKE} clean-e2e

kill-e2e:
	[ -f "e2e_api.pid" ]
	kill `cat e2e_api.pid`
	rm "e2e_api.pid"

clean-e2e: kill-e2e
	[ "$$KEEP_LOG" ] || rm "e2e_api.log"
	dropdb -U postgres miniflux_test

clean-e2e-error: kill-e2e
	false
