# Git Ignore 配置指南

## 重要提醒

本项目的 `.gitignore` 文件已经配置了以下规则来防止敏感文件被提交：

### AI 助手相关文件（绝对不应提交）

```gitignore
# AI Assistant files
.claude/
.claude*
claude.json
.anthropic/
.cursor/
.cursorignore
.aider*
.copilot/
```

这些文件包含：
- AI 会话历史
- API 密钥或令牌
- 个人配置信息
- 对话上下文

### 其他敏感文件

```gitignore
# Environment files
*.env
neo_ports.env

# Local development notes
*.local.md
notes/
.notes/

# Personal configuration
.personal/
*.personal.*
```

## 检查命令

在提交前，使用以下命令检查是否有敏感文件：

```bash
# 检查是否有 AI 相关文件
git status --ignored | grep -E "(claude|anthropic|cursor|aider|copilot)"

# 查看将要提交的文件
git status

# 如果不小心添加了敏感文件，移除它们
git rm --cached .claude
git rm --cached -r .claude/
```

## 最佳实践

1. **定期检查 .gitignore**
   - 确保新的敏感文件类型被添加到忽略列表

2. **使用 git status --ignored**
   - 查看哪些文件被忽略了

3. **不要强制添加被忽略的文件**
   - 避免使用 `git add -f` 添加被忽略的文件

4. **检查历史记录**
   - 如果敏感文件已经被提交，需要从历史中清除：
   ```bash
   git filter-branch --force --index-filter \
     "git rm --cached --ignore-unmatch .claude" \
     --prune-empty --tag-name-filter cat -- --all
   ```

## 额外的安全措施

1. **创建全局 gitignore**
   ```bash
   # 创建全局 gitignore
   echo ".claude*" >> ~/.gitignore_global
   echo ".anthropic/" >> ~/.gitignore_global
   
   # 配置 git 使用全局 gitignore
   git config --global core.excludesfile ~/.gitignore_global
   ```

2. **使用 pre-commit 钩子**
   创建 `.git/hooks/pre-commit` 文件：
   ```bash
   #!/bin/sh
   # 检查是否有 AI 相关文件
   if git diff --cached --name-only | grep -E "(\.claude|\.anthropic|\.cursor|\.aider)"; then
     echo "错误：检测到 AI 助手相关文件！"
     echo "这些文件不应该被提交到仓库。"
     exit 1
   fi
   ```

## 如果已经提交了敏感文件

如果敏感文件已经被推送到 GitHub：

1. 立即从仓库中删除
2. 更改所有相关的 API 密钥
3. 使用 GitHub 的 "Remove sensitive data" 功能
4. 考虑使用 BFG Repo-Cleaner 清理历史

记住：**预防胜于治疗**，始终在提交前仔细检查！