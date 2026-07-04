# Context Over Control Skill Refactor Goal Prompt

```text
角色：你是一个负责维护 agent skill 和 CI workflow 的资深工程师。你的任务是在 /Users/bef0rewind/Projects/work/codexsdk-go 中重构 codexsdk upstream sync skill，使它更符合 context over control：提供清晰上下文、稳定工具和安全边界，而不是把 Codex 绑死在长流程里。

# 目标

重构 `.agents/skills/codexsdk-sync-upstream` 及相关 workflow prompt，让它们表达：

- 固定、可复现、可验证的动作应该由小工具或 canonical scripts 执行
- skill 提供领域上下文、输入输出约定、完成标准、危险边界和命令路由
- workflow 只固定危险状态转换，例如权限、远端副作用、PR、merge、tag
- Codex 在受约束的阶段内自由选择最短安全路径，而不是机械执行一条长流程
- composition examples 只是建议组合，不是所有调用都必须执行的全局流程

# 先读取

开始前读取：

- `AGENTS.md`
- `.agents/skills/codexsdk-sync-upstream/SKILL.md`
- `.agents/skills/codexsdk-sync-upstream/commands/*.md`
- `.agents/skills/codexsdk-sync-upstream/references/*.md`
- `.github/prompts/codexsdk-upstream-sync-repair.md`
- `.github/workflows/upstream-protocol-auto-sync.yml`
- `scripts/codexsdk_write_sync_prompt.py`
- `scripts/codexsdk_write_sync_prompt_test.py`

先运行 `git status --short`，识别并保留与本任务无关的 dirty files。

# 重构方向

将 skill 调整为三层模型：

1. `SKILL.md` 是 contract 和 router
   - 保留领域合同、source of truth、completion layers、non-negotiable invariants
   - 列出可用 command 和固定工具
   - 说明每个 command 适合什么状态
   - composition examples 只能作为常见组合，不能暗示必须完整跑一遍

2. `commands/*.md` 是 bounded command contract
   - 每个 command 做一件事
   - 声明 inputs、outputs、allowed side effects、forbidden side effects、validation、stop rules
   - 允许 Codex 在 command 边界内自由选择读取哪些文件、跑哪些 focused checks、如何修复

3. scripts/workflow 是固定工具和危险状态门禁
   - 可复现、机械、易错、可由 exit code 判断的动作留给 scripts
   - workflow 负责 credentials、remote side effects、protected branch、PR、merge、tag
   - Codex 不应自行决定 push/tag/merge，除非当前 command 明确允许且 workflow/user 拥有该副作用

# 具体要求

检查现有 command 文件，确保它们不是隐藏的长流程脚本。每个 command 应该回答：

- 当前 command 解决什么状态
- 它信任哪些输入
- 它可以调用哪些固定工具
- 它可以改变什么
- 它绝不能改变什么
- 何时停止
- 如何验证
- 最终输出什么

把 “先 A 再 B 再 C” 的硬性流程改写为：

- 当前状态
- 可用工具
- 成功标准
- 安全边界
- 最短安全路径原则

如果现有文本把 composition examples 写成强制流程，请改成 “common compositions” 或 “typical compositions”，并明确：调用方可以直接调用单个 command，单个 command 的边界优先。

# 工具原则

当一个动作满足这些条件时，应视为固定工具或 canonical script：

- 输入输出稳定
- 行为可测试
- 成功/失败可由 exit code 判断
- 不需要大量语义判断
- 多个流程会复用
- 手写容易错或出错代价高

不要新增 orchestration framework。只有在发现一个明确缺失、固定且低风险的小工具时，才可以添加最小脚本，并同时添加测试。否则只更新 skill/prompt 文档。

# Codex 自由组合原则

Codex 应该自由处理这些判断型任务：

- 当前状态已经完成到哪一层
- 需要读哪些 compact evidence
- drift 是否需要本地 repair
- validation failure 属于生成器、schema、manifest、coverage 还是 SDK 问题
- 应该运行哪些 focused checks
- recovery 应该进入哪个 narrow command

但 Codex 不应该自由决定这些危险动作：

- push 到 protected branch
- 绕过 required checks
- 创建 PAT/App token/bot bypass
- 合成 required status
- 移动或删除 tag
- 关闭 issue 并声称 drift fully resolved
- 直接 merge 或绕开 PR 路径

# Workflow Prompt 要求

更新 `.github/prompts/codexsdk-upstream-sync-repair.md`，让它体现：

- 当前 command 是 `repair-applied-candidate`
- detect/apply 已经完成
- candidate artifacts 和 apply summary 是权威输入
- 不要重新 resolve/detect/track/apply/full Rust schema generation
- 给出 success criteria、validation expectations、final output shape
- 不规定具体读文件顺序，让 Codex 在边界内自行选择最短安全路径

# 验证

完成后运行：

```sh
python3 -m unittest scripts/codexsdk_write_sync_prompt_test.py
git diff --check -- .agents/skills/codexsdk-sync-upstream .github/prompts .github/workflows scripts/codexsdk_write_sync_prompt.py scripts/codexsdk_write_sync_prompt_test.py
ruby -e 'require "yaml"; ARGV.each { |p| YAML.load_file(p); puts "ok #{p}" }' .github/workflows/*.yml
```

如果修改了脚本，运行对应脚本测试。不要运行完整 Go validation，除非改动触及 Go 代码、生成器或 sync 脚本行为。

# 约束

- 保留无关用户改动
- 不修改 generated schema、generated Go、baseline metadata、drift reports 或 `goals/`，除非它们与本重构直接相关
- 不执行网络、GitHub issue、branch、commit、push、tag、PR、merge 操作
- 保持 ASCII，除非目标文件已有非 ASCII 风格
- 不把一次 refactor 扩展成新自动化系统

# 成功标准

完成时应满足：

- skill 更像 context/router，而不是 monolithic procedure
- command 文件表达 bounded capability，而不是强制长流程
- scripts 被描述为固定工具，agent judgment 被限制在判断型任务
- workflow prompt 给足上下文和边界，但不微管理 Codex 的内部步骤
- common compositions 是示例，不是全局强制顺序
- 测试和格式检查通过，或 blocker 被精确说明

# 最终输出

用简洁中文汇报：

- 改了哪些文件
- 如何体现 context over control
- 哪些固定工具/脚本被保留为 deterministic capability
- 哪些判断留给 Codex 自由组合
- 验证命令和结果
- 保留了哪些无关 dirty files
- 是否有 blocker 或后续风险

不要声称完成 upstream baseline sync；本任务只重构 skill/prompt/workflow contract。
```
