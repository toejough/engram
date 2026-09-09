.PHONY: validate

validate:
	@echo "Validating sensor registry..."
	@test -f lib/sensors/registry.txt || (echo "ERROR: registry.txt missing" && exit 1)
	@test -d migrations || (echo "ERROR: migrations dir missing" && exit 1)
	@ls migrations/*_sensor_*.go >/dev/null 2>&1 || (echo "ERROR: no migration files found" && exit 1)
	@grep -q "Added sensor" TELEMETRY_CHANGELOG.log || (echo "ERROR: changelog missing entry" && exit 1)
	@echo "Validation passed"
