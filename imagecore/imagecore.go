// Package imagecore 提供图像处理核心逻辑（不依赖 GUI，可独立测试）。
//
// 流程：
//  1. 任意图片（含彩色）-> 转为灰度图（标准亮度公式）
//  2. 按阈值二值化：灰度 >= 阈值 -> 白(255)，否则 -> 黑(0)
//
// 阈值用 0~100 的百分比表示，映射到 0~255 的灰度值。
package imagecore

import (
	"image"
	"image/draw"

	xdraw "golang.org/x/image/draw"
)

// PercentToThreshold 把 0~100 的百分比映射为 0~255 的灰度阈值（四舍五入）。
func PercentToThreshold(percent int) uint8 {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	// (percent*255 + 50) / 100 实现四舍五入：50%->128, 100%->255, 0%->0
	return uint8((percent*255 + 50) / 100)
}

// ToGray 把图片（可能是彩色）转换为灰度图。
//
// Go 的 color.GrayModel 使用标准亮度公式：
//
//	Y = (R*299 + G*587 + B*114) / 1000
//
// 这正是"彩色照片先转黑白"的标准做法。
// 对带透明通道的图片，先用白色背景合成，避免透明区域变黑。
func ToGray(img image.Image) *image.Gray {
	b := img.Bounds()

	// 先把图片画到白色背景上（处理透明通道），再转灰度
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, image.NewUniform(image.White), image.Point{}, draw.Src)
	draw.Draw(rgba, b, img, b.Min, draw.Over)

	gray := image.NewGray(b)
	draw.Draw(gray, b, rgba, b.Min, draw.Src) // 转换到 Gray 色彩模型
	return gray
}

// Binarize 对灰度图做二值化，返回只含纯黑(0)/纯白(255)的灰度图。
// 规则：灰度值 >= 阈值 -> 255(白)，灰度值 < 阈值 -> 0(黑)。
func Binarize(gray *image.Gray, percent int) *image.Gray {
	t := PercentToThreshold(percent)
	out := image.NewGray(gray.Bounds())
	// out 与 gray 的边界一致，Pix 布局逐一对应，可直接按下标处理
	for i, v := range gray.Pix {
		if v >= t {
			out.Pix[i] = 255
		} else {
			out.Pix[i] = 0
		}
	}
	return out
}

// BinarizeAA 是二值化的「抗锯齿」版本：在阈值附近保留过渡灰度，
// 使斜线、曲线的边缘平滑，避免锯齿（台阶状）。
//
// edgeWidth 是过渡带宽度（以灰阶为单位）：
//   - 灰度值离阈值越远 -> 越接近纯黑或纯白
//   - 落在阈值 ±edgeWidth/2 范围内 -> 按比例取过渡灰度
//
// edgeWidth 越大，边缘越柔和（但太大整体会发灰）；越小越接近纯黑白。
func BinarizeAA(gray *image.Gray, percent int, edgeWidth float64) *image.Gray {
	if edgeWidth < 1 {
		edgeWidth = 1
	}
	t := float64(PercentToThreshold(percent))

	// 预先算好 0~255 每个灰阶的映射结果（查找表，速度快）
	var lut [256]uint8
	for i := 0; i < 256; i++ {
		v := (float64(i)-t)/edgeWidth + 0.5
		switch {
		case v <= 0:
			lut[i] = 0
		case v >= 1:
			lut[i] = 255
		default:
			lut[i] = uint8(v*255 + 0.5)
		}
	}

	out := image.NewGray(gray.Bounds())
	for idx, g := range gray.Pix {
		out.Pix[idx] = lut[g]
	}
	return out
}

// ScaleDown 把灰度图等比缩小到不超过 maxW x maxH（高质量重采样）。
// 若图片本身已小于上限，则原样返回。
func ScaleDown(gray *image.Gray, maxW, maxH int) *image.Gray {
	b := gray.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxW && h <= maxH {
		return gray
	}
	// 计算缩放比例（取较小者保证两边都不超限）
	scaleW := float64(maxW) / float64(w)
	scaleH := float64(maxH) / float64(h)
	scale := scaleW
	if scaleH < scale {
		scale = scaleH
	}
	nw := int(float64(w) * scale)
	nh := int(float64(h) * scale)
	if nw < 1 {
		nw = 1
	}
	if nh < 1 {
		nh = 1
	}

	// 用 CatmullRom 高质量缩放，预览不会因缩放本身产生额外锯齿
	out := image.NewGray(image.Rect(0, 0, nw, nh))
	xdraw.CatmullRom.Scale(out, out.Bounds(), gray, gray.Bounds(), xdraw.Src, nil)
	return out
}
