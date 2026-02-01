enable_testing()

add_executable(test_header
    tests/test_header.c
)

target_link_libraries(test_header
    v6coin
    OpenSSL::Crypto
)

add_test(NAME header_test COMMAND test_header)
