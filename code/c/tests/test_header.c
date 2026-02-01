#include "v6_coin_header.h"
#include <stdio.h>
#include <assert.h>
#include <string.h>

void test_header_init_and_validate() {
    printf("Testing header initialization and validation...\n");

    V6CoinExtHeader header;
    uint8_t sender[16] = {0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01};
    uint8_t receiver[16] = {0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x02};

    v6_coin_header_init(&header, TX_TYPE_ONLINE, sender, receiver, 1000, 1640000000);

    assert(header.option_type == 0x7F);
    assert(header.version == V6_COIN_VERSION);
    assert(header.tx_type == TX_TYPE_ONLINE);
    assert(header.amount == 1000);

    int valid = v6_coin_header_validate(&header);
    assert(valid == 1);

    printf("✓ Header initialization and validation passed\n");
}

void test_header_serialize_deserialize() {
    printf("Testing header serialization and deserialization...\n");

    V6CoinExtHeader original;
    uint8_t sender[16] = {0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01};
    uint8_t receiver[16] = {0x20, 0x01, 0x0d, 0xb8, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x02};

    v6_coin_header_init(&original, TX_TYPE_OFFLINE, sender, receiver, 5000, 1640000000);

    uint8_t buffer[256];
    size_t serialized_len = v6_coin_header_serialize(&original, buffer, sizeof(buffer));
    assert(serialized_len > 0);

    V6CoinExtHeader deserialized;
    size_t deserialized_len = v6_coin_header_deserialize(buffer, serialized_len, &deserialized);
    assert(deserialized_len == serialized_len);

    assert(memcmp(original.sender_addr, deserialized.sender_addr, 16) == 0);
    assert(memcmp(original.receiver_addr, deserialized.receiver_addr, 16) == 0);
    assert(original.amount == deserialized.amount);
    assert(original.tx_type == deserialized.tx_type);

    v6_coin_header_free(&deserialized);

    printf("✓ Header serialization and deserialization passed\n");
}

void test_header_validation_failures() {
    printf("Testing header validation failures...\n");

    V6CoinExtHeader header;
    v6_coin_header_init(&header, TX_TYPE_ONLINE, (uint8_t[16]){0}, (uint8_t[16]){0}, 0, 0);

    // Invalid option type
    header.option_type = 0xFF;
    assert(v6_coin_header_validate(&header) == 0);

    // Invalid version
    header.option_type = 0x7F;
    header.version = 0x0000;
    assert(v6_coin_header_validate(&header) == 0);

    // Invalid transaction type
    header.version = V6_COIN_VERSION;
    header.tx_type = 0xFF;
    assert(v6_coin_header_validate(&header) == 0);

    printf("✓ Header validation failures passed\n");
}

int main() {
    printf("Running V6Coin Header Tests\n");
    printf("==============================\n\n");

    test_header_init_and_validate();
    test_header_serialize_deserialize();
    test_header_validation_failures();

    printf("\nAll tests passed! ✓\n");
    return 0;
}
