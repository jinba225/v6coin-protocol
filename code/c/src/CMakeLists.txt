add_library(v6coin STATIC
    v6_coin_header.c
)

target_link_libraries(v6coin
    OpenSSL::Crypto
)
