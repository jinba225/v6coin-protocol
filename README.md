# V6Coin Protocol | V6币协议

> 这只是一个想法，希望能被看见、实现。｜ This is just an idea, I hope it can be seen and realized.

IPv6 IoT Network Native Value Incentive Layer | IPv6物联网原生价值激励层

Production-Grade Technical Whitepaper & Open Protocol Specification | 生产级技术白皮书+开源协议规范

**语言切换 | Language Switch**

[中文版本](#CN)|[English Version](#EN)

---

# CN
## ✨ 核心亮点

✅ **原生IPv6协议耦合**：128位地址天然实现「地址即钱包」，无需额外映射表，完美契合物联网设备原生网络属性  

✅ **绿色共识机制**：PoC（连接证明）替代PoW，无能源浪费；比PoS更公平，前缀反垄断机制防止中心化垄断  

✅ **极致兼容性**：IPv6扩展报头+UDP/ICMP退化封装，适配防火墙、低版本路由器等现实网络环境  

✅ **民主化治理**：贡献度加权投票制，长期提供网络服务的「劳动节点」拥有60%话语权，杜绝巨鲸操纵  

✅ **合规隐私平衡**：可选审计观察位设计，满足企业级合规需求，同时不破坏全局匿名性底座  

✅ **成本稳定机制**：流量价格锚定机制隔绝币价波动，保障开发者使用网络服务的实际成本长期可控  

## 📚 快速导航

- 📄 [完整白皮书](docs/whitepaper/V6Coin_Whitepaper_CN.md)

- 📖 [技术附录规范](docs/specs/cn/)

- 🔧 [可复用代码片段](code/)

- 🎁 [开发者激励计划](bounty/Bounty_Program_CN.md)

- 🌱 [种子节点计划](bounty/Seed_Node_Plan_CN.md)

- 🤝 [贡献指南](CONTRIBUTING.md)

## 🔑 核心技术参数

| 参数名称           | 取值                                               |
| ------------------ | -------------------------------------------------- |
| IPv6扩展报头类型   | Hop-by-Hop Options Header (0x7F)                   |
| 加密算法           | Ed25519签名 / AES-256加密 / SHA-256哈希            |
| 共识机制           | PoC (Proof of Connection)                          |
| 区块间隔           | 10秒                                               |
| 交易确认延迟       | ≤30秒（3个区块间隔）                               |
| 最小单位           | 1 nano-V6 (10⁻⁹ V6)                                |
| 总量硬顶           | 830亿枚                                            |
| 投票权重分配       | PoC贡献度(60%) + 持币量(40%)，单节点持币权重上限5% |
| 离线交易有效期     | 7天                                                |
| 最大逻辑子网前缀数 | 8个                                                |

## 🎯 激励计划

### 开发者赏金

| 激励类型 | 奖励范围                                                     |
| -------- | ------------------------------------------------------------ |
| 漏洞赏金 | 低危：1-10万V6<br>中危：10-50万V6<br>高危：50-200万V6<br>致命：200-1000万V6 |
| 系统适配 | OpenWrt/OpenHarmony等IoT系统适配：50-200万V6/款              |
| 硬件集成 | 第三方网关/传感器集成：100-300万V6/款                        |
| 代码贡献 | 有效PR合并：1-50万V6/次                                      |

### 种子节点计划

- 全球限量：1000个首批稳定节点

- 核心权益：挖矿奖励系数+1.2倍（永久），优先成为验证节点

- 参与条件：连续在线≥99%，有效转发量≥100GB/天，无作恶记录

## 🛠️ 开发者快速上手

### 1. 核心代码获取

```Bash
# 克隆仓库
git clone https://github.com/jinba225/v6coin-protocol.git
cd v6coin-protocol

# 查看核心协议代码（C/Go双语言）
cd code/c  # C语言头文件（IPv6报头结构体，直接复用）
cd code/go # Go语言结构体（物联网开发首选）
```

### 2. 协议接入关键点

- 地址生成：通过Ed25519私钥推导64位IID，组合IPv6前缀形成唯一钱包地址

- 交易发起：支持在线（扩展报头嵌入）/离线（本地签名缓存）双模式

- 兼容性适配：自动检测网络环境，不支持扩展报头时自动切换UDP(38901端口)封装

- 密钥管理：遵循BIP-39助记词标准，支持无芯片IoT设备安全激活与资产迁移

### 3. 测试网络参与

关注仓库「Issues」板块，获取测试网激活码与节点部署指南，首批参与者可额外获得5万V6激励

## 📜 开源协议

本项目基于 **MIT开源协议** 授权 - 详见 [LICENSE](LICENSE) 文件，允许自由使用、修改、商用，无需额外授权（保留版权声明即可）。

## 🤝 贡献指南

欢迎所有开发者参与协议优化、代码开发、文档完善！

- 提交Bug/建议：通过GitHub Issues提交（标题格式：[类型] 问题描述）

- 提交代码：Fork仓库 → 新建分支 → 提交PR → 审核合并（合并后按贡献度发放激励）

- 所有贡献者将被记录在「贡献者名单」，永久留存于仓库

## 📞 联系与社区

- GitHub：[https://github.com/jinba225/v6coin-protocol](https://github.com/jinba225/v6coin-protocol)



# EN
## ✨ Core Highlights

✅ **Native IPv6 Integration**：128-bit address natively enables "Address is Wallet" without additional mapping tables, perfectly matching the native network attributes of IoT devices  

✅ **Green Consensus Mechanism**：PoC (Proof of Connection) replaces PoW with no energy waste; fairer than PoS, with prefix anti-monopoly mechanism to prevent centralization  

✅ **Extreme Compatibility**：IPv6 extension header + UDP/ICMP degradation encapsulation, compatible with real network environments such as firewalls and low-version routers  

✅ **Democratic Governance**：Contribution-weighted voting system, "labor nodes" providing long-term network services have 60% voting power to prevent whale manipulation  

✅ **Compliance & Privacy Balance**：Optional Audit Observer Bit design meets enterprise-level compliance needs without damaging the global anonymity base  

✅ **Stable Cost Control**：Traffic Price Anchoring Mechanism isolates token price fluctuations, ensuring long-term controllable actual costs of network services for developers  

## 📚 Quick Navigation

- 📄 [Full Whitepaper](docs/whitepaper/V6Coin_Whitepaper_EN.md)

- 📖 [Technical Specifications](docs/specs/en/)

- 🔧 [Reusable Code Snippets](code/)

- 🎁 [Developer Bounty Program](bounty/Bounty_Program_EN.md)

- 🌱 [Seed Node Plan](bounty/Seed_Node_Plan_EN.md)

- 🤝 [Contribution Guidelines](CONTRIBUTING.md)

## 🔑 Core Technical Specs

| Parameter Name                 | Value                                                        |
| ------------------------------ | ------------------------------------------------------------ |
| IPv6 Extension Header Type     | Hop-by-Hop Options Header (0x7F)                             |
| Encryption Algorithm           | Ed25519 Signature / AES-256 Encryption / SHA-256 Hashing     |
| Consensus Mechanism            | PoC (Proof of Connection)                                    |
| Block Interval                 | 10 Seconds                                                   |
| Transaction Confirmation Delay | ≤30 Seconds (3 Block Intervals)                              |
| Minimum Unit                   | 1 nano-V6 (10⁻⁹ V6)                                          |
| Total Supply Cap               | 83 Billion V6                                                |
| Voting Weight Distribution     | PoC Contribution (60%) + Token Holding (40%), max 5% voting weight per node |
| Offline Transaction Validity   | 7 Days                                                       |
| Max Logical Subnet Prefixes    | 8                                                            |

## 🎯 Incentive Program

### Developer Bounty

| Incentive Type       | Reward Range                                                 |
| -------------------- | ------------------------------------------------------------ |
| Bug Bounty           | Low-Risk: 10k-100k V6<br>Medium-Risk: 100k-500k V6<br>High-Risk: 500k-2M V6<br>Critical: 2M-10M V6 |
| System Adaptation    | IoT OS (OpenWrt/OpenHarmony) Adaptation: 500k-2M V6/OS       |
| Hardware Integration | Third-party Gateway/Sensor Integration: 1M-3M V6/Device      |
| Code Contribution    | Valid PR Merged: 10k-500k V6/PR                              |

### Seed Node Program

- Global Quota: 1000 Initial Stable Nodes

- Core Benefits: 1.2x Mining Reward Multiplier (Permanent), Priority as Validation Node

- Eligibility: ≥99% Uptime, ≥100GB/Day Valid Forwarding, No Malicious Behavior

## 🛠️ Quick Start for Developers

### 1. Core Code Access

```Bash
# Clone Repository
git clone https://github.com/jinba225/v6coin-protocol.git
cd v6coin-protocol

# View Core Protocol Code (C/Go Dual-Language)
cd code/c  # C Header File (IPv6 Header Structure, Ready-to-Use)
cd code/go # Go Structure (Preferred for IoT Development)
```

### 2. Key Integration Points

- Address Generation: Derive 64-bit IID via Ed25519 private key, combine with IPv6 prefix to form unique wallet address

- Transaction Initiation: Supports dual modes - Online (embedded in extension header) / Offline (local signature cache)

- Compatibility Adaptation: Automatically detects network environment, switches to UDP (Port 38901) encapsulation when extension headers are not supported

- Key Management: Complies with BIP-39 mnemonic standard, supporting secure activation and asset migration for chip-free IoT devices

### 3. Testnet Participation

Follow the 「Issues」section for testnet activation codes and node deployment guides. Early participants receive an additional 50k V6 incentive.

## 📜 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details. Free to use, modify, and commercialize without additional authorization (retain copyright notice).

## 🤝 Contribution Guidelines

We welcome all developers to participate in protocol optimization, code development, and documentation improvement!

- Submit Bugs/Suggestions: Via GitHub Issues (Title Format: [Type] Issue Description)

- Submit Code: Fork Repository → Create Branch → Submit PR → Review & Merge (Incentives distributed based on contribution after merging)

- All contributors will be listed in the 「Contributor List」permanently in the repository

## 📞 Contact & Community

- GitHub: [https://github.com/jinba225/v6coin-protocol](https://github.com/jinba225/v6coin-protocol)

</div>

---

🌟 V6 Coin - 让每个IPv6地址都成为价值节点 | Let Every IPv6 Address Become a Value Node 🌟

🚀 推动物联网从「信息互联」迈向「价值互联」 | Promote IoT from "Information Interconnection" to "Value Interconnection" 🚀
