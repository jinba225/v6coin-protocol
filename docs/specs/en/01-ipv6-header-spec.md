# IPv6 Extension Header Specification

## Version Information

- **Version**: v1.0
- **Last Updated**: 2026-02-01
- **Protocol Version**: 0x0100

## Overview

The V6Coin protocol implements direct transaction transmission via IPv6 extension headers, eliminating the need for additional encapsulation layers. This approach leverages native IPv6 features to embed transaction data into the network layer header, realizing the "Address is Wallet" philosophy.

### Design Goals

1. **Native Integration**: V6Coin transactions are directly part of the IPv6 header without requiring additional protocol layers
2. **Efficient Transmission**: Utilizes the mandatory processing characteristic of Hop-by-Hop Options Header to ensure all nodes can identify and process transactions
3. **High Compatibility**: Provides fallback encapsulation for network environments that don't support extension headers
4. **QoS Support**: Leverages IPv6 Traffic Class fields for quality of service classification

## Extension Header Type

### Hop-by-Hop Options Header (0x7F)

V6Coin uses the **Hop-by-Hop Options Header** as the extension header type for the following reasons:

1. **Mandatory Processing**: According to RFC 8200, all intermediate nodes must process Hop-by-Hop Options, ensuring V6Coin transactions are identified by every node
2. **Hop-by-Hop Delivery**: Suitable for scenarios requiring inspection and processing by each node along the transmission path
3. **Flexible Extension**: Supports custom Option Types, enabling future expansion

### Option Type Definition

```
V6Coin Option Type: 0x7F
- 0x7F: V6Coin transaction data option (reserved for experimental use)
```

## Data Structure Definitions

### IPv6 Base Header Structure (40 bytes)

```
+-----+-----+-----+-----+-----+-----+-----+-----+
|  0  |  1  |  2  |  3  |  4  |  5  |  6  |  7  |
+-----+-----+-----+-----+-----+-----+-----+-----+
| Version | Traffic Class |           Flow Label          |
+---------+---------------+------------------------------+
|        Payload Length   | Next Header |    Hop Limit     |
+------------------------+------------------+-----------+
|                    Source Address                          |
+                                                               +
|                                                               +
+---------------------------------------------------------------+
|                   Destination Address                         |
+                                                               +
|                                                               +
+---------------------------------------------------------------+
```

**Field Descriptions**:
- `Version`: 4 bits, IPv6 version number (fixed at 6)
- `Traffic Class`: 8 bits, traffic classification (DSCP + ECN)
- `Flow Label`: 20 bits, flow label
- `Payload Length`: 16 bits, payload length
- `Next Header`: 8 bits, next header type (Hop-by-Hop Options = 0)
- `Hop Limit`: 8 bits, hop limit
- `Source Address`: 128 bits, source IPv6 address
- `Destination Address`: 128 bits, destination IPv6 address

### Hop-by-Hop Options Header Structure

```
+-----+-----+-----+-----+-----+-----+-----+-----+
| Next Header | Hdr Ext Len |       Option 1       |
+-------------+-------------+---------------------+
| Option Type | Opt Data Len |   Option Data       |
+-------------+-------------+---------------------+
|             ... Additional Options ...      |
+----------------------------------------------+
```

**Field Descriptions**:
- `Next Header`: 8 bits, next header type
- `Hdr Ext Len`: 8 bits, extension header length (in 8-octet units, excluding the first 8 octets)
- `Option Type`: 8 bits, option type (V6Coin = 0x7F)
- `Opt Data Len`: 8 bits, option data length
- `Option Data`: variable length, option data

### V6Coin Transaction Structure

```
+-----+-----+-----+-----+-----+-----+-----+-----+
|  Version   |TxType|         Timestamp         |
+-----+-----+-----+-----------------------------------+
|                  Sender Address (16 bytes)     |
+                                               +
|                                               +
+-----------------------------------------------+
|                 Receiver Address (16 bytes)   |
+                                               +
|                                               +
+-----------------------------------------------+
|                  Amount (8 bytes)             |
+-----------------------------------------------+
|                                               |
|             Signature (32 bytes)             |
|                                               |
|                                               +
+-----------------------------------------------+
|                  Reserved (optional)         |
+-----------------------------------------------+
```

**Minimum Length**: 83 bytes (excluding Option Type and Opt Data Len fields)

## V6Coin Transaction Format Details

### 1. Version (Protocol Version)

- **Length**: 2 bytes
- **Encoding**: Big-Endian
- **Value**: 0x0100 (v1.0)
- **Description**: V6Coin protocol version number for compatibility checking

### 2. TxType (Transaction Type)

- **Length**: 1 byte
- **Values**:
  - `0x01`: Online Transaction - real-time confirmation
  - `0x02`: Offline Transaction - delayed confirmation
  - `0x03`: Migration - cross-prefix asset migration

### 3. Timestamp

- **Length**: 8 bytes
- **Encoding**: Big-Endian
- **Unit**: Unix timestamp (seconds)
- **Description**: Transaction creation time for replay attack prevention and transaction ordering

### 4. SenderAddr (Sender Address)

- **Length**: 16 bytes
- **Format**: IPv6 address (128 bits)
- **Description**: Sender's V6Coin wallet address, which is their IPv6 address
- **Encoding**: Raw binary format

### 5. RecvAddr (Receiver Address)

- **Length**: 16 bytes
- **Format**: IPv6 address (128 bits)
- **Description**: Receiver's V6Coin wallet address
- **Encoding**: Raw binary format

### 6. Amount

- **Length**: 8 bytes
- **Encoding**: Big-Endian
- **Unit**: nano-V6 (10⁻⁹ V6)
- **Range**: 0 ~ 18,446,744,073,709,551,615 nano-V6
- **Description**: Transaction amount, minimum unit is nano-V6

### 7. Signature

- **Length**: 32 bytes
- **Algorithm**: Ed25519
- **Description**: Signature of transaction data (all fields except Signature field) using sender's private key
- **Verification**: Receiver verifies signature validity using sender's public key

### 8. Reserved

- **Length**: Optional, variable
- **Description**: Reserved field for future expansion
- **Default Value**: 0 (if present)

## Packet Encapsulation Structure

### Complete Packet Structure

```
+-------------------------------------------------------+
|                  IPv6 Header (40 bytes)              |
|  Version(4) | Traffic Class(8) | Flow Label(20)       |
|  Payload Length | Next Header(0) | Hop Limit          |
|  Source Address (16 bytes)                            |
|  Destination Address (16 bytes)                       |
+-------------------------------------------------------+
|            Hop-by-Hop Options Header                  |
|  Next Header | Hdr Ext Len                            |
|  Option Type (0x7F) | Opt Data Len                    |
|  V6Coin Transaction Data (83+ bytes)                  |
+-------------------------------------------------------+
```

### Packet Length Calculation

```
Minimum Total Length = 40 (IPv6 Header) + 8 (Hop-by-Hop Base) + 2 (Option Fields) + 83 (Tx Data) + Padding
                     = 133 bytes + Padding (pad to 8-byte boundary)
                     = 136 bytes
```

## UDP/ICMP Fallback Encapsulation

### Use Cases

When the network environment doesn't support IPv6 extension headers (e.g., firewall filtering, older routers), V6Coin automatically switches to fallback encapsulation mode.

### UDP Encapsulation Format

**Port**: 38901

```
+-------------------------------------------------------+
|                   UDP Header (8 bytes)                |
|  Source Port (16) | Dest Port (16)                   |
|  Length (16)      | Checksum (16)                    |
+-------------------------------------------------------+
|            V6Coin Transaction Data                    |
|  Version | TxType | Timestamp | ...                  |
+-------------------------------------------------------+
```

### ICMP Encapsulation Format

**Type**: Dedicated V6Coin ICMP Type (to be assigned)

```
+-------------------------------------------------------+
|                   ICMP Header (8 bytes)               |
|  Type (8) | Code (8) | Checksum (16)                  |
|  Identifier (16) | Sequence (16)                     |
+-------------------------------------------------------+
|            V6Coin Transaction Data                    |
+-------------------------------------------------------+
```

### Fallback Strategy

1. **Auto Detection**: Sender detects network extension header support via ICMP error messages or timeouts
2. **Retry on Failure**: Automatically switch to UDP encapsulation after extension header send failure
3. **Fallback Indication**: Receiver confirms actual encapsulation method via response header type

## QoS Support

### Traffic Class Field Structure

```
+-----+-----+-----+-----+-----+-----+-----+-----+
|  0  |  1  |  2  |  3  |  4  |  5  |  6  |  7  |
+-----+-----+-----+-----+-----+-----+-----+-----+
|           DSCP (6 bits)        | ECN (2 bits)|
+----------------------------------------------+
```

### DSCP Mapping (QoS Levels)

| QoS Level     | DSCP Value | Purpose                | Priority |
|---------------|------------|------------------------|----------|
| Urgent        | 46 (EF)    | System critical tx     | Highest  |
| High Priority | 34 (AF41)  | Asset migration       | High     |
| Normal        | 0 (BE)     | Regular transaction    | Medium   |
| Low Priority  | 8 (CS1)    | Bulk verification      | Low      |

### ECN Mapping (Explicit Congestion Notification)

| ECN Value | Meaning       | Description                          |
|-----------|---------------|--------------------------------------|
| 00        | Non-ECT       | ECN not supported                    |
| 01        | ECT(1)        | ECN supported, in transit            |
| 10        | ECT(0)        | ECN supported, in transit            |
| 11        | CE            | Congestion experienced, reduce rate  |

### QoS Setting Example

```go
// Set high priority QoS
dscp := uint8(34)  // AF41
ecn := uint8(0)    // ECT(0)
header.TrafficClass = ((dscp & 0x3F) << 2) | (ecn & 0x03)
```

## Compatibility Handling

### Network Environment Detection

1. **Extension Header Support Test**: Send test packet with Hop-by-Hop Options
2. **Response Analysis**: Detect ICMP Parameter Problem error messages
3. **Fallback Mechanism**: Automatically switch to UDP encapsulation when not supported

### Fallback Strategy Priority

1. **Priority 1**: IPv6 Extension Header (native mode)
2. **Priority 2**: UDP Encapsulation (port 38901)
3. **Priority 3**: ICMP Encapsulation (last resort)

### Compatibility Matrix

| Network Type        | Extension Header | UDP Encapsulation | ICMP Encapsulation |
|---------------------|------------------|-------------------|--------------------|
| Pure IPv6 Network   | ✅               | ✅                | ✅                 |
| NAT66 Network       | ⚠️               | ✅                | ✅                 |
| Firewall Strict     | ❌               | ✅                | ✅                 |
| IPv4/IPv6 Dual Stack | ❌               | ✅                | ✅                 |

## Code Examples

### Go Example: Create V6Coin Extension Header

```go
package main

import (
    "encoding/binary"
    "net"
    "v6coin-protocol/pkg/ipv6"
)

// Create V6Coin transaction
func createV6CoinTransaction(sender, receiver string, amount uint64) *ipv6.V6CoinTransaction {
    return &ipv6.V6CoinTransaction{
        Version:    0x0100,
        TxType:     0x01, // Online transaction
        Timestamp:  uint64(1725148800), // 2024-09-01 00:00:00
        SenderAddr: net.ParseIP(sender),
        RecvAddr:   net.ParseIP(receiver),
        Amount:     amount,
        Signature:  make([]byte, 32), // Need to sign with private key
    }
}

// Encapsulate in IPv6 extension header
func encapsulateInIPv6Header(source, dest string, tx *ipv6.V6CoinTransaction) ([]byte, error) {
    // Create IPv6 base header
    header := &ipv6.IPv6Header{
        Version:       6,
        TrafficClass:  0, // DSCP=0, ECN=0
        FlowLabel:     0,
        NextHeader:    ipv6.NextHeaderHopByHopOptions, // 0
        HopLimit:      64,
        SourceAddr:    net.ParseIP(source),
        DestAddr:      net.ParseIP(dest),
    }

    // Create V6Coin extension header
    extHeader, err := ipv6.NewV6CoinExtensionHeader(tx)
    if err != nil {
        return nil, err
    }

    // Calculate payload length
    hbhData, _ := ipv6.SerializeHopByHopOptions(&extHeader.HopByHop)
    header.PayloadLength = uint16(len(hbhData))

    // Serialize IPv6 header
    headerData, err := ipv6.SerializeIPv6Header(header)
    if err != nil {
        return nil, err
    }

    // Assemble complete packet
    packet := append(headerData, hbhData...)
    return packet, nil
}

// Set QoS
func setQoS(header *ipv6.IPv6Header, dscp, ecn uint8) {
    header.TrafficClass = ipv6.BuildTrafficClass(dscp, ecn)
}
```

### Packet Structure Diagram

```
Send Example: 2001:db8::1 → 2001:db8::2, transfer 1000000 nano-V6

0000: 60 00 00 00 00 58 00 40 2001 0db8 0000 0000 0000 0000
0020: 0000 0001 2001 0db8 0000 0000 0000 0000 0000 0002 00
0040: 10 7F 53 01 00 00 00 66 d2 90 00 00 2001 0db8 0000 0000
0060: 0000 0000 0000 0001 2001 0db8 0000 0000 0000 0000 0000
0080: 0002 00 00 00 00 00 0F 42 40 00 00 00 00 00 00 00 00 00
00A0: 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00 00

Parse:
- IPv6 Header: 40 bytes
  - Next Header: 0x00 (Hop-by-Hop Options)
  - Payload Length: 0x0058 (88 bytes)
  - Source: 2001:db8::1
  - Dest: 2001:db8::2

- Hop-by-Hop Options Header:
  - Next Header: 0x00 (None)
  - Hdr Ext Len: 0x10 (16 * 8 = 128 bytes, including first 8 bytes = 136 bytes total)
  - Option Type: 0x7F (V6Coin)
  - Opt Data Len: 0x53 (83 bytes)

- V6Coin Transaction:
  - Version: 0x0100
  - TxType: 0x01 (Online)
  - Timestamp: 0x00000066D2900000 (1725148800)
  - Sender: 2001:db8::1
  - Receiver: 2001:db8::2
  - Amount: 0x00000000000F4240 (1000000 nano-V6)
  - Signature: 0x00... (empty signature for example)
```

## Validation Rules

### Transaction Validation

1. **Version Check**: Version must equal 0x0100
2. **Transaction Type**: TxType must be in range 0x01 ~ 0x03
3. **Address Format**: SenderAddr and RecvAddr must be valid IPv6 addresses (16 bytes)
4. **Signature Verification**: Signature must be 32 bytes and pass Ed25519 verification
5. **Timestamp**: Timestamp must be within reasonable range (current time ± 24 hours)

### Extension Header Validation

1. **Option Type**: Must equal 0x7F
2. **Data Length**: Opt Data Len + 83 must equal actual data length
3. **Padding Check**: Total extension header length must be a multiple of 8 bytes

## Security Considerations

### Signature Protection

- All transactions must be signed using the Ed25519 algorithm
- Signature covers all transaction data except the Signature field
- Receiver must verify the validity of the sender's public key

### Replay Attack Protection

- Each transaction contains a unique timestamp
- Nodes maintain a transaction hash cache for the last 24 hours
- Duplicate timestamp + same sender = reject transaction

### Privacy Protection

- IPv6 address is the wallet address, naturally anonymous
- Optional audit observer bit design allows partial transaction traceability
- No additional identity information required to complete transactions

## References

- [RFC 8200: IPv6 Specification](https://datatracker.ietf.org/doc/html/rfc8200)
- [RFC 8200: IPv6 Extension Headers](https://datatracker.ietf.org/doc/html/rfc8200#section-4)
- [RFC 3168: The Addition of Explicit Congestion Notification (ECN) to IP](https://datatracker.ietf.org/doc/html/rfc3168)
- [RFC 8085: UDP Usage Guidelines](https://datatracker.ietf.org/doc/html/rfc8085)
- [Ed25519: High-speed high-security signatures](https://ed25519.cr.yp.to/)

## Version History

| Version | Date       | Change Description                       |
|---------|------------|-------------------------------------------|
| v1.0    | 2026-02-01 | Initial version, defining basic structures |

## Appendix

### A. Byte Order Explanation

- All multi-byte fields use Big-Endian network byte order
- Ensures consistency in data exchange across different platforms

### B. Error Code Definitions

| Error Code | Meaning                  | Handling Action  |
|------------|--------------------------|-------------------|
| 0x01       | Version mismatch         | Reject transaction |
| 0x02       | Invalid transaction type | Reject transaction |
| 0x03       | Address format error     | Reject transaction |
| 0x04       | Signature verification failed | Reject transaction |
| 0x05       | Invalid timestamp        | Reject transaction |
| 0x06       | Extension header format error | Drop packet   |

### C. Constant Definitions

```go
const (
    V6CoinVersion      = 0x0100
    V6CoinOptionType   = 0x7F
    TxTypeOnline       = 0x01
    TxTypeOffline      = 0x02
    TxTypeMigration    = 0x03
    MaxTxDataLen       = 83  // Minimum transaction data length
    SignatureLen       = 32  // Ed25519 signature length
    IPv6AddrLen        = 16  // IPv6 address length
    UDPPort            = 38901  // V6Coin UDP encapsulation port
)
```
