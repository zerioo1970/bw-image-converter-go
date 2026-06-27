package imagecore

import (
	"image"
	"image/color"
	"testing"
)

func TestPercentToThreshold(t *testing.T) {
	cases := map[int]uint8{0: 0, 50: 128, 100: 255}
	for p, want := range cases {
		if got := PercentToThreshold(p); got != want {
			t.Fatalf("PercentToThreshold(%d)=%d, 期望 %d", p, got, want)
		}
	}
	// 越界保护
	if PercentToThreshold(-5) != 0 || PercentToThreshold(150) != 255 {
		t.Fatal("越界百分比未被夹紧")
	}
}

func TestBinarizeGradient(t *testing.T) {
	// 造一张 256x1 的水平灰度渐变（0..255）
	grad := image.NewGray(image.Rect(0, 0, 256, 1))
	for i := 0; i < 256; i++ {
		grad.SetGray(i, 0, color.Gray{Y: uint8(i)})
	}

	out := Binarize(grad, 50) // 阈值=128
	for i := 0; i < 256; i++ {
		v := out.GrayAt(i, 0).Y
		if i < 128 && v != 0 {
			t.Fatalf("像素 %d 应为黑(0)，实为 %d", i, v)
		}
		if i >= 128 && v != 255 {
			t.Fatalf("像素 %d 应为白(255)，实为 %d", i, v)
		}
	}
}

func TestColorToGray(t *testing.T) {
	// 偏红的颜色 (200,50,50) 亮度 ~ 0.299*200+0.587*50+0.114*50 ≈ 95
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 50, B: 50, A: 255})
		}
	}
	gray := ToGray(img)
	g := gray.GrayAt(0, 0).Y
	if g < 90 || g > 100 {
		t.Fatalf("红色块灰度值 %d 不在预期范围(约95)", g)
	}
	// 95 < 128 -> 二值化应为黑
	if Binarize(gray, 50).GrayAt(0, 0).Y != 0 {
		t.Fatal("偏红色块在50%阈值下应为黑")
	}
}

func TestExtremes(t *testing.T) {
	grad := image.NewGray(image.Rect(0, 0, 256, 1))
	for i := 0; i < 256; i++ {
		grad.SetGray(i, 0, color.Gray{Y: uint8(i)})
	}
	// 0% 阈值=0 -> 全白
	out0 := Binarize(grad, 0)
	for i := 0; i < 256; i++ {
		if out0.GrayAt(i, 0).Y != 255 {
			t.Fatal("0% 阈值应全白")
		}
	}
	// 100% 阈值=255 -> 仅 255 为白
	out100 := Binarize(grad, 100)
	if out100.GrayAt(255, 0).Y != 255 || out100.GrayAt(254, 0).Y != 0 {
		t.Fatal("100% 阈值行为不正确")
	}
}

func TestBinarizeAA(t *testing.T) {
	// 渐变图，AA 二值化应：远离阈值的为纯黑/纯白，阈值附近出现过渡灰
	grad := image.NewGray(image.Rect(0, 0, 256, 1))
	for i := 0; i < 256; i++ {
		grad.SetGray(i, 0, color.Gray{Y: uint8(i)})
	}
	out := BinarizeAA(grad, 50, 40) // 阈值=128, 过渡带宽=40

	// 远低于阈值（如灰度0）应为纯黑
	if out.GrayAt(0, 0).Y != 0 {
		t.Fatalf("远低于阈值应为纯黑，实为 %d", out.GrayAt(0, 0).Y)
	}
	// 远高于阈值（如灰度255）应为纯白
	if out.GrayAt(255, 0).Y != 255 {
		t.Fatalf("远高于阈值应为纯白，实为 %d", out.GrayAt(255, 0).Y)
	}
	// 阈值处(128)应为中间灰，且严格介于黑白之间（即存在抗锯齿过渡）
	mid := out.GrayAt(128, 0).Y
	if mid == 0 || mid == 255 {
		t.Fatalf("阈值附近应有过渡灰度，实为 %d", mid)
	}
	// 单调不减：灰度递增，输出也应递增（或持平）
	prev := -1
	for i := 0; i < 256; i++ {
		v := int(out.GrayAt(i, 0).Y)
		if v < prev {
			t.Fatalf("AA 输出应单调不减，位置 %d 出现下降", i)
		}
		prev = v
	}
}

func TestWhitenAbove(t *testing.T) {
	grad := image.NewGray(image.Rect(0, 0, 256, 1))
	for i := 0; i < 256; i++ {
		grad.SetGray(i, 0, color.Gray{Y: uint8(i)})
	}
	out := WhitenAbove(grad, 50) // 阈值=128
	for i := 0; i < 256; i++ {
		v := out.GrayAt(i, 0).Y
		if i > 128 {
			// 高于阈值 -> 纯白
			if v != 255 {
				t.Fatalf("灰度 %d 高于阈值，应为纯白(255)，实为 %d", i, v)
			}
		} else {
			// 不高于阈值 -> 原样不变
			if int(v) != i {
				t.Fatalf("灰度 %d 未高于阈值，应保持不变，实为 %d", i, v)
			}
		}
	}
}

func TestScaleDown(t *testing.T) {
	big := image.NewGray(image.Rect(0, 0, 1000, 500))
	out := ScaleDown(big, 200, 200)
	if out.Bounds().Dx() > 200 || out.Bounds().Dy() > 200 {
		t.Fatalf("缩小后尺寸超限：%v", out.Bounds())
	}
	// 等比：1000x500 限制到 200x200，宽先到 200，则高=100
	if out.Bounds().Dx() != 200 || out.Bounds().Dy() != 100 {
		t.Fatalf("等比缩放结果不对：%v", out.Bounds())
	}
	// 小图原样返回
	small := image.NewGray(image.Rect(0, 0, 50, 50))
	if ScaleDown(small, 200, 200) != small {
		t.Fatal("小于上限的图应原样返回")
	}
}
