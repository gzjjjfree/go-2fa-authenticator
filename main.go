package main

import (
	"encoding/json"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/pquerna/otp/totp"
)

// 定义全局的密钥存储 Map
var (
	secrets map[string]string
)

func main() {
	// 1. 初始化 Fyne App (设置一个唯一的 App ID，用于本地 Preferences 存储)
	myApp := app.NewWithID("com.myname.2fa_authenticator")
	myWindow := myApp.NewWindow("2FA 认证器")
	myWindow.Resize(fyne.NewSize(400, 500))

	// 2. 加载本地存储的密钥
	secrets = loadSecrets(myApp)

	// ================= UI 组件 =================

	// 验证码显示标签
	codeLabel := widget.NewLabelWithStyle("------", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	codeLabel.SizeName = theme.SizeNameWindowTitleBarHeight //.TextSize = 48

	// 30秒倒计时进度条
	progressBar := widget.NewProgressBar()
	progressBar.Max = 30
	progressBar.SetValue(30)

	// 账户下拉选择框
	var accountSelect *widget.Select
	accountNames := getAccountNames()

	accountSelect = widget.NewSelect(accountNames, func(selected string) {
		updateCode(selected, codeLabel)
	})
	accountSelect.PlaceHolder = "请选择一个账号..."

	// 添加新账户的表单
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("例如: Google 或 GitHub")
	secretEntry := widget.NewEntry()
	secretEntry.SetPlaceHolder("输入 16/32 位纯文本 Secret")

	addButton := widget.NewButtonWithIcon("保存新账号", theme.DocumentSaveIcon(), func() {
		name := nameEntry.Text
		secret := secretEntry.Text

		if name == "" || secret == "" {
			dialog.ShowInformation("错误", "名称和密钥不能为空", myWindow)
			return
		}

		// 测试密钥是否合法
		_, err := totp.GenerateCode(secret, time.Now())
		if err != nil {
			dialog.ShowError(fmt.Errorf("无效的密钥格式: %v", err), myWindow)
			return
		}

		// 保存并刷新下拉列表
		secrets[name] = secret
		saveSecrets(myApp, secrets)

		accountSelect.Options = getAccountNames()
		accountSelect.SetSelected(name) // 自动选中刚添加的项

		nameEntry.SetText("")
		secretEntry.SetText("")
		dialog.ShowInformation("成功", "账号已保存", myWindow)
	})

	// ================= 布局组装 =================

	topSection := container.NewVBox(
		widget.NewLabel("选择账号:"),
		accountSelect,
	)

	centerSection := container.NewVBox(
		codeLabel,
		progressBar,
	)

	bottomSection := container.NewVBox(
		widget.NewSeparator(),
		widget.NewLabelWithStyle("添加新账号", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nameEntry,
		secretEntry,
		addButton,
	)

	// 使用 Border 布局，让核心代码部分居中
	content := container.NewBorder(topSection, bottomSection, nil, nil, container.NewCenter(centerSection))
	myWindow.SetContent(content)

	// ================= 后台定时刷新 =================

	go func() {
		for {
			time.Sleep(1 * time.Second)

			// Fyne 的 UI 更新是线程安全的
			if accountSelect.Selected != "" {
				updateCode(accountSelect.Selected, codeLabel)
			}

			// 计算当前时间在 30 秒周期内的剩余时间
			now := time.Now().Unix()
			remain := 30 - (now % 30)
			fyne.Do(func() { progressBar.SetValue(float64(remain)) })
		}
	}()

	myWindow.ShowAndRun()
}

// ---------------- 辅助函数 ----------------

// 加载本地配置
func loadSecrets(a fyne.App) map[string]string {
	data := a.Preferences().String("secrets_data")
	res := make(map[string]string)
	if data != "" {
		_ = json.Unmarshal([]byte(data), &res)
	}
	return res
}

// 保存配置到本地
func saveSecrets(a fyne.App, data map[string]string) {
	bytes, _ := json.Marshal(data)
	a.Preferences().SetString("secrets_data", string(bytes))
}

// 获取所有账户名称列表
func getAccountNames() []string {
	names := make([]string, 0, len(secrets))
	for k := range secrets {
		names = append(names, k)
	}
	return names
}

// 更新验证码 UI
func updateCode(accountName string, label *widget.Label) {
	secret, exists := secrets[accountName]
	if !exists {
		fyne.Do(func() { label.SetText("------") })
		return
	}

	code, err := totp.GenerateCode(secret, time.Now())
	if err == nil {
		// 在中间加个空格，视觉上更像 Authenticator (例如 "123 456")
		formattedCode := code[:3] + " " + code[3:]
		fyne.Do(func() { label.SetText(formattedCode) })
	} else {
		fyne.Do(func() { label.SetText("Error") })
	}
}
