---
name: sensors-add-skill
description: Use when registering or adding a new sensor type to the telemetry system's registry, codegen, migrations, and changelog
---

# Add Sensor Type to Telemetry System

Register a new sensor type in the telemetry system following the project's established conventions.

## Procedure

1. Add a new entry to lib/sensors/registry.txt in tab-separated format: <sensor_id>:<version>\t<sensor_name>\t<date>. Example line: `pressure_v2:1.0\tpressure\t2024-02-01`

2. Run the codegen script to update type definitions: `python3 scripts/sensors.py`

3. Create a migration file in migrations/ with naming pattern NNNN_sensor_<name>.go where NNNN is the next 4-digit sequence number (the first migration is 0001; use the next 4-digit number after the highest existing one). The migration must contain a func init() block that calls registerSensor(<sensor_id>).

4. Add an entry to TELEMETRY_CHANGELOG.log with exact format: `[YYYY-MM-DD HH:MM:SS] <operator> Added sensor <id>:<version>`

5. Verify all changes are staged in git (do not commit yet).

6. Run `make validate` to verify the sensor registration is complete and correct.
