package imgserver

import (
	log "github.com/Sirupsen/logrus"
	"github.com/disintegration/imaging"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path"
	"path/filepath"
	"strconv"
)

func Watermark(imgpath string, watermarkpath string) error {
	file_ext := path.Ext(imgpath)

	//原始图片
	imgb, err := os.Open(imgpath)
	if err != nil {
		return err
	}
	defer imgb.Close()

	var img image.Image

	if file_ext == ".jpg" || file_ext == ".jpeg" {
		img, err = jpeg.Decode(imgb)
	} else if file_ext == ".png" {
		img, err = png.Decode(imgb)
	} else {
		//except jpg/png of other image don't add watermark
		return nil
	}
	if err != nil {
		return err
	}

	wmb, err1 := os.Open(watermarkpath)
	if err1 != nil {
		return err1
	}
	defer wmb.Close()
	watermark, err2 := png.Decode(wmb)
	if err2 != nil {
		return err2
	}
	log.Debugf("image height : " + strconv.Itoa(img.Bounds().Dy()))

	waterSize := int(math.Floor(float64(img.Bounds().Dy() / 10)))
	waterOffset := waterSize / 10
	log.Debugf("image water offset : %s", strconv.Itoa(waterOffset))

	watermark = imaging.Resize(watermark, waterSize, waterSize, imaging.Lanczos)

	//把水印写到右下角，并向0坐标各偏移10个像素
	offset := image.Pt(img.Bounds().Dx()-watermark.Bounds().Dx()-waterOffset, img.Bounds().Dy()-watermark.Bounds().Dy()-waterOffset)
	b := img.Bounds()
	m := image.NewNRGBA(b)

	draw.Draw(m, b, img, image.ZP, draw.Src)
	draw.Draw(m, watermark.Bounds().Add(offset), watermark, image.ZP, draw.Over)

	//覆盖图片
	imgw, _ := os.Create(imgpath)

	jpeg.Encode(imgw, m, &jpeg.Options{100})

	log.Infof("watermark success - %s", filepath.Base(imgpath))

	defer imgw.Close()

	return nil
}
