import behavioral


def test_exit_code_assert():
    assert behavioral.assert_on(7, "", "exit:7") is True
    assert behavioral.assert_on(1, "", "exit:7") is False
    # a non-integer exit spec must NOT raise — the try/except int() returns False
    assert behavioral.assert_on(1, "", "exit:notanumber") is False
