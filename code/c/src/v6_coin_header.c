#include "v6_coin_header.h"
#include <string.h>
#include <assert.h>

// Initialize V6Coin extension header
void v6_coin_header_init(V6CoinExtHeader *header, V6CoinTxType tx_type,
                         const uint8_t *sender, const uint8_t *receiver,
                         uint64_t amount, uint64_t timestamp) {
    assert(header != NULL);
    assert(sender != NULL);
    assert(receiver != NULL);

    header->option_type = 0x7F;
    header->version = V6_COIN_VERSION;
    header->tx_type = tx_type;
    header->timestamp = timestamp;
    header->amount = amount;

    memcpy(header->sender_addr, sender, 16);
    memcpy(header->receiver_addr, receiver, 16);

    memset(header->auth_signature, 0, 32);
}

// Validate V6Coin extension header
int v6_coin_header_validate(const V6CoinExtHeader *header) {
    if (header == NULL) return 0;
    if (header->option_type != 0x7F) return 0;
    if (header->version != V6_COIN_VERSION) return 0;
    if (header->tx_type < TX_TYPE_ONLINE || header->tx_type > TX_TYPE_MIGRATION) return 0;

    return 1;
}

// Serialize V6Coin extension header to buffer
size_t v6_coin_header_serialize(const V6CoinExtHeader *header, uint8_t *buffer, size_t buffer_size) {
    size_t total_len = v6_coin_header_total_len(header);
    if (buffer_size < total_len) return 0;

    size_t offset = 0;
    memcpy(buffer + offset, &header->option_type, sizeof(header->option_type));
    offset += sizeof(header->option_type);

    memcpy(buffer + offset, &header->opt_data_len, sizeof(header->opt_data_len));
    offset += sizeof(header->opt_data_len);

    memcpy(buffer + offset, &header->version, sizeof(header->version));
    offset += sizeof(header->version);

    memcpy(buffer + offset, &header->tx_type, sizeof(header->tx_type));
    offset += sizeof(header->tx_type);

    memcpy(buffer + offset, &header->timestamp, sizeof(header->timestamp));
    offset += sizeof(header->timestamp);

    memcpy(buffer + offset, header->sender_addr, 16);
    offset += 16;

    memcpy(buffer + offset, header->receiver_addr, 16);
    offset += 16;

    memcpy(buffer + offset, &header->amount, sizeof(header->amount));
    offset += sizeof(header->amount);

    memcpy(buffer + offset, header->auth_signature, 32);
    offset += 32;

    if (header->opt_data_len > 0) {
        size_t reserved_len = header->opt_data_len - 0x26; // Subtract fixed fields length
        memcpy(buffer + offset, header->reserved, reserved_len);
    }

    return total_len;
}

// Deserialize V6Coin extension header from buffer
size_t v6_coin_header_deserialize(const uint8_t *buffer, size_t buffer_size, V6CoinExtHeader *header) {
    if (buffer == NULL || header == NULL) return 0;
    if (buffer_size < 0x2C) return 0; // Minimum length

    size_t offset = 0;
    memcpy(&header->option_type, buffer + offset, sizeof(header->option_type));
    offset += sizeof(header->option_type);

    memcpy(&header->opt_data_len, buffer + offset, sizeof(header->opt_data_len));
    offset += sizeof(header->opt_data_len);

    memcpy(&header->version, buffer + offset, sizeof(header->version));
    offset += sizeof(header->version);

    memcpy(&header->tx_type, buffer + offset, sizeof(header->tx_type));
    offset += sizeof(header->tx_type);

    memcpy(&header->timestamp, buffer + offset, sizeof(header->timestamp));
    offset += sizeof(header->timestamp);

    memcpy(header->sender_addr, buffer + offset, 16);
    offset += 16;

    memcpy(header->receiver_addr, buffer + offset, 16);
    offset += 16;

    memcpy(&header->amount, buffer + offset, sizeof(header->amount));
    offset += sizeof(header->amount);

    memcpy(header->auth_signature, buffer + offset, 32);
    offset += 32;

    if (header->opt_data_len > 0x26) {
        size_t reserved_len = header->opt_data_len - 0x26;
        if (buffer_size < offset + reserved_len) return 0;
        header->reserved = (uint8_t*)malloc(reserved_len);
        memcpy(header->reserved, buffer + offset, reserved_len);
    }

    return v6_coin_header_total_len(header);
}

// Free resources allocated during deserialization
void v6_coin_header_free(V6CoinExtHeader *header) {
    if (header != NULL && header->reserved != NULL) {
        free(header->reserved);
        header->reserved = NULL;
    }
}
