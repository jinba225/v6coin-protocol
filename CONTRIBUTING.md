# Contributing to V6Coin Protocol

感谢你考虑为 V6Coin Protocol 做出贡献！我们欢迎所有形式的贡献，包括但不限于代码、文档、bug报告、功能建议等。

## 行为准则

- 尊重他人，保持友好和专业的态度
- 欢迎新成员，帮助他们适应
- 专注于对社区最有利的事情
- 对不同观点保持开放心态

## 如何贡献

### 报告 Bug

如果你发现了 bug，请创建一个 Issue，包含以下信息：

- Bug 的详细描述
- 重现步骤
- 预期行为
- 实际行为
- 环境信息（操作系统、Go/C 版本等）
- 相关的日志或截图

### 提交功能请求

如果你有新功能的想法，请创建一个 Issue，说明：

- 功能的描述和用例
- 为什么这个功能有用
- 你可能想到的实现方案

### 提交代码

#### 1. Fork 仓库

点击右上角的 "Fork" 按钮，将仓库 fork 到你的 GitHub 账号。

#### 2. 克隆仓库

```bash
git clone https://github.com/YOUR_USERNAME/v6coin-protocol.git
cd v6coin-protocol
```

#### 3. 创建分支

为你的贡献创建一个新分支：

```bash
git checkout -b feature/your-feature-name
# 或
git checkout -b fix/your-bug-fix
```

#### 4. 进行开发

按照项目的代码规范进行开发，确保代码风格一致。

##### Go 代码规范

- 遵循 [Effective Go](https://golang.org/doc/effective_go) 和 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` 格式化代码
- 添加必要的注释和文档
- 编写单元测试，覆盖率 > 80%

```bash
# 格式化代码
go fmt ./...

# 运行测试
go test -v -race -cover ./...

# 运行 linter
golangci-lint run
```

##### C 代码规范

- 遵循 Google C Style Guide
- 使用一致的命名约定
- 添加必要的注释和文档
- 编写单元测试

```bash
# 构建和测试
cd code/c
cmake -B build
cmake --build build
ctest --test-dir build
```

#### 5. 提交变更

提交你的变更，使用清晰的提交信息：

```bash
git add .
git commit -m "feat: add PoC consensus mechanism"
```

提交信息格式：
- `feat:` 新功能
- `fix:` Bug 修复
- `docs:` 文档更新
- `style:` 代码格式调整
- `refactor:` 代码重构
- `test:` 测试相关
- `chore:` 构建/工具相关

#### 6. 推送分支

```bash
git push origin feature/your-feature-name
```

#### 7. 创建 Pull Request

- 访问原仓库的 "Pull requests" 页面
- 点击 "New pull request"
- 选择你的分支
- 填写 PR 模板，详细描述你的变更
- 等待代码审查

### 文档贡献

文档同样重要！你可以：

- 改进现有文档的清晰度和准确性
- 添加新的教程和示例
- 翻译文档到其他语言
- 添加代码注释

文档位置：
- `doc/` - 技术文档
- `code/sdk/examples/` - 示例代码
- 代码内注释

## 开发环境设置

### Go 开发环境

```bash
# 安装 Go 1.21+
# 从 https://golang.org/dl/ 下载安装

# 克隆仓库
git clone https://github.com/jinba225/v6coin-protocol.git
cd v6coin-protocol

# 下载依赖
cd code/go
go mod download

# 运行测试
go test ./...

# 运行代码检查
golangci-lint run
```

### C 开发环境

```bash
# 安装依赖 (Ubuntu/Debian)
sudo apt-get update
sudo apt-get install build-essential cmake libssl-dev

# 安装依赖 (macOS)
brew install cmake openssl

# 构建项目
cd code/c
cmake -B build
cmake --build build

# 运行测试
ctest --test-dir build
```

## 代码审查

所有的 Pull Request 都需要经过代码审查才能合并。审查过程中，我们可能会：

- 提出改进建议
- 要求添加测试
- 要求修改代码风格
- 要求更新文档

请积极配合审查者，及时回应反馈。

## 变更日志

重要的变更会在 `CHANGELOG.md` 中记录。如果你修改了公共 API 或行为，请更新变更日志。

## 激励计划

我们通过 Bounty 计划激励贡献者：

- Bug 修复：1-50 万 V6
- 功能实现：5-200 万 V6
- 文档改进：1-20 万 V6
- 测试用例：1-10 万 V6

详见 `bounty/Bounty_Program_CN.md`。

## 许可证

通过提交代码，你同意你的代码将按照项目的 MIT 许可证进行授权。

## 获取帮助

如果你有任何问题，可以：

- 创建 Issue 寻求帮助
- 在 Discussions 中讨论
- 查阅项目文档

## 社区

- GitHub: https://github.com/jinba225/v6coin-protocol
- Issues: https://github.com/jinba225/v6coin-protocol/issues
- Discussions: https://github.com/jinba225/v6coin-protocol/discussions

感谢你的贡献！让我们一起构建更好的 V6Coin Protocol！
