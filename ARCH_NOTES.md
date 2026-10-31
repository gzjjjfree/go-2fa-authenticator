# 2FA 认证器项目架构与修改记录

## 核心架构设计
- **框架**: 使用 `fyne.io/fyne/v2` 构建跨平台桌面图形界面。
- **OTP 算法**: 使用 `github.com/pquerna/otp/totp` 生成和验证 TOTP 动态验证码。
- **本地持久化**: 采用 Fyne `App.Preferences()` 存储以 JSON 序列化形式保存的账号映射表 (`map[string]string`)。

## 功能更新记录 (当前版本)
1. **同窗口 Tab 标签页架构 (`container.NewAppTabs`)**:
   - 将主界面重构为多标签页布局，包含“验证码”标签页和“账号管理”标签页，无需弹出独立窗口。
2. **账号查看、编辑与删除功能**:
   - **账号管理页**: 在“账号管理”标签中展示所有已保存账号列表及操作按钮。
   - **全宽度内联编辑弹窗 (`dialog.NewCustomConfirm`)**: 点击编辑时在原窗口内弹出表单对话框，宽度自动适配为窗口宽度减 20px（左右两边沿各留 10px 间距），支持修改账号名称及密钥，并自动进行 TOTP 格式校验。
   - **删除确认**: 支持确认删除账号并同步刷新选择器与主界面。
3. **CI/CD 发布流水线扩充 (`release.yml`)**:
   - 在 GitHub Actions 中新增 `build-windows` 任务（基于 `windows-latest` 运行环境使用 `fyne package -os windows`），自动编译并生成 Windows `.exe` 可执行文件，并在 `create-release` 阶段自动上传至 GitHub Release 附件中。
