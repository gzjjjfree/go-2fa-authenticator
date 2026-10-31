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
	codeLabel.SizeName = theme.SizeNameWindowTitleBarHeight

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

	// ================= 主页布局 =================

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

	homeContent := container.NewBorder(topSection, bottomSection, nil, nil, container.NewCenter(centerSection))

	// ================= 账号管理 Tab 内容 =================
	var manageListContainer *container.Scroll
	var refreshManageList func()

	refreshManageList = func() {
		vbox := container.NewVBox()
		if len(secrets) == 0 {
			vbox.Add(widget.NewLabel("暂无保存的账号"))
		} else {
			for name, secret := range secrets {
				accName := name
				accSecret := secret

				nameLabel := widget.NewLabelWithStyle(accName, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

				editBtn := widget.NewButton("编辑", func() {
					showEditDialogInWindow(myWindow, accName, accSecret, func(newName, newSecret string) {
						if newName != accName {
							if _, exists := secrets[newName]; exists {
								dialog.ShowInformation("错误", "该账号名称已存在", myWindow)
								return
							}
							delete(secrets, accName)
						}
						secrets[newName] = newSecret
						saveSecrets(myApp, secrets)
						accountSelect.Options = getAccountNames()
						accountSelect.Refresh()
						if accountSelect.Selected == accName || accountSelect.Selected == newName {
							accountSelect.SetSelected(newName)
						}
						refreshManageList()
					})
				})

				delBtn := widget.NewButton("删除", func() {
					dialog.ShowConfirm("确认删除", fmt.Sprintf("确定要删除账号 '%s' 吗？", accName), func(ok bool) {
						if ok {
							delete(secrets, accName)
							saveSecrets(myApp, secrets)
							accountSelect.Options = getAccountNames()
							accountSelect.Refresh()
							if accountSelect.Selected == accName {
								accountSelect.SetSelected("")
								codeLabel.SetText("------")
							}
							refreshManageList()
						}
					}, myWindow)
				})

				row := container.NewBorder(nil, nil, nameLabel, container.NewHBox(editBtn, delBtn))
				vbox.Add(row)
				vbox.Add(widget.NewSeparator())
			}
		}
		manageListContainer.Content = vbox
		manageListContainer.Refresh()
	}

	manageListContainer = container.NewVScroll(container.NewVBox())
	refreshManageList()

	manageContent := container.NewBorder(widget.NewLabelWithStyle("已保存账号列表", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), nil, nil, nil, manageListContainer)

	tabs := container.NewAppTabs(
		container.NewTabItem("验证码", homeContent),
		container.NewTabItem("账号管理", container.NewPadded(manageContent)),
	)

	tabs.OnSelected = func(ti *container.TabItem) {
		if ti.Text == "账号管理" {
			refreshManageList()
		}
	}

	myWindow.SetContent(tabs)

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

// 在原窗口内显示编辑账号弹窗 (Dialog)，宽度设为屏幕宽度减 20px
func showEditDialogInWindow(parentWindow fyne.Window, currentName string, currentSecret string, onSave func(string, string)) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(currentName)

	secretEntry := widget.NewEntry()
	secretEntry.SetText(currentSecret)

	formContainer := container.NewVBox(
		widget.NewLabel("账号名称:"),
		nameEntry,
		widget.NewLabel("密钥 (Secret):"),
		secretEntry,
	)

	d := dialog.NewCustomConfirm("编辑账号: "+currentName, "保存", "取消", container.NewPadded(formContainer), func(ok bool) {
		if !ok {
			return
		}
		newName := nameEntry.Text
		newSecret := secretEntry.Text

		if newName == "" || newSecret == "" {
			dialog.ShowInformation("错误", "名称和密钥不能为空", parentWindow)
			return
		}

		_, err := totp.GenerateCode(newSecret, time.Now())
		if err != nil {
			dialog.ShowError(fmt.Errorf("无效的密钥格式: %v", err), parentWindow)
			return
		}

		onSave(newName, newSecret)
	}, parentWindow)

	winSize := parentWindow.Canvas().Size()
	dialogWidth := winSize.Width - 20
	if dialogWidth < 280 {
		dialogWidth = 380
	}
	d.Resize(fyne.NewSize(dialogWidth, 220))
	d.Show()
}
