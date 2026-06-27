# 黑白分明图片生成器（Go 独立 exe 版）

把照片转成**只有纯黑和纯白**两种颜色的图片。彩色照片会先转灰度，再按你设定的灰度阈值二值化。

本版本用 **Go** 编写，编译成**单个独立的 Windows `.exe`，双击即可运行，无需安装 Python、.NET 或任何运行时**。

## 直接使用（最简单，无需编译）

1. 打开仓库里的 **`黑白图片生成器.exe`**
2. 点 **Download raw file**（下载按钮）把它下载到电脑
3. **双击运行**即可

> 这是一个静态编译的独立程序，拷到任何 Windows（64 位）上都能直接跑，不依赖任何外部库。

## 使用方法

1. 点「选择图片」按钮挑选图片（**支持中文文件名**）
2. 拖动滑块（0% ~ 100%）设定**灰度阈值**
   - 灰度值 **≥ 阈值** → 转为**纯白**
   - 灰度值 **< 阈值** → 转为**纯黑**
3. 拖动滑块时**实时预览**效果
4. 勾选/取消 **「平滑边缘（抗锯齿）」**：
   - **勾选**（默认）：斜线、曲线边缘平滑，无锯齿，整体仍是黑白分明（边缘保留少量过渡灰）
   - **取消**：纯黑白两色，斜线会有锯齿（台阶状）
5. 点「保存到桌面」，结果图(PNG)会保存到电脑桌面（文件名为 `原名_黑白.png`）

> 阈值百分比映射到 0~255 灰度值，例如 50% = 灰度值 128。

### 关于斜线锯齿

纯黑白二值化时，每个像素非黑即白，斜线/曲线无法平滑过渡，会呈"台阶状"锯齿。
开启「平滑边缘」后，程序只在**线条边缘**保留少量过渡灰度，肉眼看去斜线变平滑，
整体观感仍是黑白分明。若坚持要绝对纯两色，可关闭该选项（但锯齿不可避免）。
另外，**源图分辨率越高，锯齿越不明显**。

## 自己编译（可选）

需要安装 [Go](https://go.dev/dl/)。

**在 Windows 上编译：**
```bash
go build -ldflags "-H=windowsgui -s -w" -o 黑白图片生成器.exe .
```

**在 Linux / macOS 上交叉编译出 Windows exe：**
```bash
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "-H=windowsgui -s -w" -o 黑白图片生成器.exe .
```

> 修改 GUI 后若需要重新生成 Windows 资源文件（含应用清单），可用 rsrc：
> ```bash
> go install github.com/akavel/rsrc@latest
> rsrc -manifest app.manifest -arch amd64 -o rsrc_windows_amd64.syso
> ```

## 项目结构

| 文件 | 作用 |
|---|---|
| `main.go` | 程序主入口，walk 图形界面 |
| `imagecore/imagecore.go` | 图像处理核心（转灰度 + 阈值二值化 + 缩放），不依赖界面 |
| `imagecore/imagecore_test.go` | 核心逻辑的 Go 测试（`go test ./imagecore`）|
| `app.manifest` | Windows 应用清单（启用现代控件样式、DPI 感知）|
| `rsrc_windows_amd64.syso` | 由清单生成的资源文件，编译时自动嵌入 |
| `黑白图片生成器.exe` | 预编译好的独立可执行文件 |

## 技术说明

- GUI：[lxn/walk](https://github.com/lxn/walk)（Windows 原生控件，纯 Go）
- 图像：Go 标准库 `image` + `golang.org/x/image`（支持 jpg/png/gif/bmp/webp）
- 灰度转换用标准亮度公式：`Y = R*0.299 + G*0.587 + B*0.114`
- 二值化用查找表，预览用缩小图保证拖动流畅，保存用原图全分辨率保证清晰
- `CGO_ENABLED=0` 静态编译，生成零依赖的独立 exe

## 已测试

图像处理核心逻辑已通过 Go 单元测试（`go test ./imagecore`）：阈值映射、渐变二值化、彩色转灰度、极端值、等比缩放。
> 注意：GUI 窗口部分需在 Windows 上实际运行查看效果。
