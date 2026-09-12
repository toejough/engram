# Widget API Specification

## Purpose

The widget API is a small internal HTTP service that lets other internal teams list and inspect widget inventory records without direct database access.

## Requirements

### Requirement: The API SHALL list widgets

The system SHALL provide a GET endpoint returning all widgets currently in inventory.

#### Scenario: List widgets
- **WHEN** a GET request is made to /widgets
- **THEN** the response is a 200 with a JSON array of widgets
