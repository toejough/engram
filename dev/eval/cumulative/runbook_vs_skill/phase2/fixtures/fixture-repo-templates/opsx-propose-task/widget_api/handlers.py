"""HTTP handlers for the widget API."""

WIDGETS = []


def list_widgets():
    """Return all widgets currently in inventory."""
    return list(WIDGETS)
