#ifndef V6_COIN_HEADER_H
#define V6_COIN_HEADER_H

#include <stdint.h>
#include <stddef.h>

// V6Coin protocol version
#define V6_COIN_VERSION 0x0100

// Transaction types
typedef enum {
    TX_TYPE_ONLINE = 0x01,    // Online real-time transaction
    TX_TYPE_OFFLINE = 0x02,   // Offline delayed transaction
    TX_TYPE_MIGRATION = 0x03  // Asset migration transaction
} V6CoinTxType;

// IPv6 extension header transaction data structure
typedef struct {
    uint8_t option_type;                // Option Type: 0x7F
    uint8_t opt_data_len;               // Opt Data Len: bytes, excluding option_type and opt_data_len
    uint16_t version;                   // V6-Coin Version: V6_COIN_VERSION
    V6CoinTxType tx_type;               // Tx Type: transaction type
    uint64_t timestamp;                 // Timestamp: Unix timestamp (seconds)
    uint8_t sender_addr[16];            // Sender Address (Compressed): compressed IPv6 address (128bit)
    uint8_t receiver_addr[16];          // Receiver Address (Compressed): compressed IPv6 address (128bit)
    uint64_t amount;                    // Amount (nano-V6): transaction amount in nano-V6, little-endian
    uint8_t auth_signature[32];         // V6-Coin Auth Signature: Ed25519 signature (256bit)
    uint8_t reserved[];                 // Reserved: reserved fields for future expansion
} V6CoinExtHeader;

// Calculate total length of extension header (including option_type and opt_data_len)
static inline size_t v6_coin_header_total_len(const V6CoinExtHeader *header) {
    return sizeof(header->option_type) + sizeof(header->opt_data_len) + header->opt_data_len;
}

#endif // V6_COIN_HEADER_H
