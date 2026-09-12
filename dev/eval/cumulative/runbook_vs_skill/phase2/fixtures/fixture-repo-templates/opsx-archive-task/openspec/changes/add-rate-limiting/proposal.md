## Why

The widget API has no rate limiting on its list endpoint, so a single client can exhaust server capacity with a burst of requests.

## What Changes

- Add a per-client rate limit to the widget API's list endpoint.

## Capabilities

### Modified Capabilities
- `widget-api`: adds a rate-limiting requirement to the existing list endpoint

## Impact

- Widget API request-handling code; no data migration required.
