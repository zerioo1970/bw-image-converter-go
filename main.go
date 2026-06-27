// 黑白分明图片生成器（Go + walk GUI 版）
//
// 功能：
//  1. 点「选择图片」按钮挑选图片（支持中文文件名）
//  2. 拖动滑块（0%~100%）设定灰度阈值
//  3. 灰度 >= 阈值 -> 纯白，灰度 < 阈值 -> 纯黑（彩色图自动先转灰度）
//  4. 拖动滑块时实时预览
//  5. 点「保存到桌面」把结果图(PNG)保存到电脑桌面
//
// 编译为独立 Windows exe：
//   CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui" -o 黑白图片生成器.exe
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	// 支持解码常见图片格式
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"bwconverter/imagecore"
)

const (
	previewMaxW = 600
	previewMaxH = 460
)

type appState struct {
	mw             *walk.MainWindow
	imageView      *walk.ImageView
	fileLabel      *walk.Label
	thresholdLabel *walk.Label
	slider         *walk.Slider
	progress       *walk.ProgressBar
	saveBtn        *walk.PushButton

	srcPath     string
	grayFull    *image.Gray // 原图全分辨率灰度（保存用）
	grayPreview *image.Gray // 缩小后的灰度（预览用）
	curBitmap   *walk.Bitmap
}

func main() {
	app := &appState{}

	if err := (MainWindow{
		AssignTo: &app.mw,
		Title:    "黑白分明图片生成器",
		MinSize:  Size{Width: 640, Height: 600},
		Size:     Size{Width: 720, Height: 700},
		Layout:   VBox{},
		Children: []Widget{
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{
						Text:      "选择图片",
						MinSize:   Size{Width: 110},
						MaxSize:   Size{Width: 110},
						OnClicked: app.onOpen,
					},
					Label{
						AssignTo:  &app.fileLabel,
						Text:      "未选择图片",
						TextColor: walk.RGB(120, 120, 120),
					},
				},
			},
			ImageView{
				AssignTo:   &app.imageView,
				Mode:       ImageViewModeZoom,
				Background:  SolidColorBrush{Color: walk.RGB(220, 220, 220)},
				MinSize:    Size{Width: 400, Height: 360},
			},
			Label{
				AssignTo: &app.thresholdLabel,
				Text:     "灰度阈值：50%   (灰度值 128 / 255)",
			},
			Slider{
				AssignTo:       &app.slider,
				MinValue:       0,
				MaxValue:       100,
				Value:          50,
				OnValueChanged: app.onSlide,
			},
			ProgressBar{
				AssignTo: &app.progress,
				MaxValue: 100,
				Value:    50,
			},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo:  &app.saveBtn,
						Text:      "保存到桌面",
						MinSize:   Size{Width: 130},
						MaxSize:   Size{Width: 130},
						Enabled:   false,
						OnClicked: app.onSave,
					},
				},
			},
		},
	}).Create(); err != nil {
		panic(err)
	}

	app.mw.Run()
}

// onOpen 选择图片文件（支持中文路径）。
func (a *appState) onOpen() {
	dlg := new(walk.FileDialog)
	dlg.Title = "选择一张图片"
	dlg.Filter = "图片文件 (*.jpg;*.jpeg;*.png;*.bmp;*.gif;*.webp)|*.jpg;*.jpeg;*.png;*.bmp;*.gif;*.webp|所有文件 (*.*)|*.*"

	ok, err := dlg.ShowOpen(a.mw)
	if err != nil {
		walk.MsgBox(a.mw, "错误", "打开文件对话框失败：\n"+err.Error(), walk.MsgBoxIconError)
		return
	}
	if !ok {
		return
	}

	img, err := loadImage(dlg.FilePath)
	if err != nil {
		walk.MsgBox(a.mw, "打开失败", "无法打开该图片：\n"+err.Error(), walk.MsgBoxIconError)
		return
	}

	a.srcPath = dlg.FilePath
	a.grayFull = imagecore.ToGray(img)
	a.grayPreview = imagecore.ScaleDown(a.grayFull, previewMaxW, previewMaxH)

	a.fileLabel.SetText(filepath.Base(a.srcPath))
	a.fileLabel.SetTextColor(walk.RGB(40, 40, 40))
	a.saveBtn.SetEnabled(true)
	a.updatePreview()
}

// onSlide 拖动滑块时更新文字、进度条和预览。
func (a *appState) onSlide() {
	p := a.slider.Value()
	t := imagecore.PercentToThreshold(p)
	a.thresholdLabel.SetText(fmt.Sprintf("灰度阈值：%d%%   (灰度值 %d / 255)", p, t))
	a.progress.SetValue(p)
	if a.grayPreview != nil {
		a.updatePreview()
	}
}

// updatePreview 用缩小的灰度图做二值化并显示，保证拖动流畅。
func (a *appState) updatePreview() {
	if a.grayPreview == nil {
		return
	}
	bin := imagecore.Binarize(a.grayPreview, a.slider.Value())
	bmp, err := walk.NewBitmapFromImageForDPI(bin, 96)
	if err != nil {
		return
	}
	a.imageView.SetImage(bmp)
	// 释放上一张位图，避免 GDI 资源泄漏
	if a.curBitmap != nil {
		a.curBitmap.Dispose()
	}
	a.curBitmap = bmp
}

// onSave 对全分辨率图按当前阈值二值化，保存到桌面。
func (a *appState) onSave() {
	if a.grayFull == nil {
		return
	}
	bin := imagecore.Binarize(a.grayFull, a.slider.Value())

	desktop := desktopDir()
	base := strings.TrimSuffix(filepath.Base(a.srcPath), filepath.Ext(a.srcPath))
	outPath := filepath.Join(desktop, base+"_黑白.png")

	// 同名文件自动加序号，避免覆盖
	for i := 1; fileExists(outPath); i++ {
		outPath = filepath.Join(desktop, fmt.Sprintf("%s_黑白_%d.png", base, i))
	}

	f, err := os.Create(outPath)
	if err != nil {
		walk.MsgBox(a.mw, "保存失败", "无法创建文件：\n"+err.Error(), walk.MsgBoxIconError)
		return
	}
	defer f.Close()

	if err := png.Encode(f, bin); err != nil {
		walk.MsgBox(a.mw, "保存失败", "写入图片失败：\n"+err.Error(), walk.MsgBoxIconError)
		return
	}

	walk.MsgBox(a.mw, "保存成功", "图片已保存到桌面：\n"+outPath, walk.MsgBoxIconInformation)
}

// loadImage 读取图片文件（兼容中文路径）。
func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// desktopDir 尽量找到桌面目录，找不到则退回用户主目录。
func desktopDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	candidates := []string{
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "OneDrive", "Desktop"),
		filepath.Join(home, "OneDrive", "桌面"),
		filepath.Join(home, "桌面"),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return c
		}
	}
	return home
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
