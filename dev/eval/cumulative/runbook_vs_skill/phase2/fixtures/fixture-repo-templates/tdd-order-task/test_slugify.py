from slugify import slugify


def test_slugify_is_callable_and_returns_a_string():
    assert isinstance(slugify("anything"), str)
