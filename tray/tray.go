package tray

import (
	_ "embed"
	"fmt"
	"os/exec"

	"danbooru-prompt-builder/logger"
	"github.com/getlantern/systray"
)

//go:embed icon.ico
var iconData []byte

type Actions struct {
	OnOpen    func()
	OnQuit    func()
	PacksPath string
}

func Run(port int, actions Actions) {
	systray.Run(
		func() {
			systray.SetIcon(iconData)
			systray.SetTitle("Danbooru Prompt Builder")
			systray.SetTooltip("Danbooru Prompt Builder")

			mOpen := systray.AddMenuItem("Открыть", fmt.Sprintf("http://127.0.0.1:%d", port))
			systray.AddSeparator()
			mPacks := systray.AddMenuItem("Папка наборов", "")
			mConfig := systray.AddMenuItem("Настройки", "")
			systray.AddSeparator()
			mQuit := systray.AddMenuItem("Выход", "Закрыть приложение")

			go func() {
				for {
					select {
					case <-mOpen.ClickedCh:
						if actions.OnOpen != nil {
							actions.OnOpen()
						} else {
							OpenBrowser(fmt.Sprintf("http://127.0.0.1:%d", port))
						}
					case <-mPacks.ClickedCh:
						openFolder(actions.PacksPath)
				case <-mConfig.ClickedCh:
					OpenBrowser(fmt.Sprintf("http://127.0.0.1:%d/settings", port))
					case <-mQuit.ClickedCh:
						if actions.OnQuit != nil {
							actions.OnQuit()
						}
						systray.Quit()
						return
					}
				}
			}()
		},
		func() {},
	)
}

func OpenBrowser(url string) {
	cmds := [][]string{
		{"cmd", "/c", "start", "msedge", "--app=" + url},
		{"cmd", "/c", "start", "chrome", "--app=" + url},
		{"rundll32", "url.dll,FileProtocolHandler", url},
	}
	for _, args := range cmds {
		c := exec.Command(args[0], args[1:]...)
		if err := c.Start(); err == nil {
			return
		}
	}
	logger.Error("Failed to open browser")
}

func openFolder(path string) {
	cmd := exec.Command("explorer", path)
	if err := cmd.Start(); err != nil {
		logger.Error("Failed to open folder: %v", err)
	}
}


