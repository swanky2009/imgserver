package imgserver

import (
	log "github.com/Sirupsen/logrus"
	"github.com/disintegration/imaging"
	"image"
	"math"
	"path"
	"path/filepath"
	"strconv"
)

func Watermark(imgpath string, watermarkpath string) error {

	file_ext := path.Ext(imgpath)

	if !(file_ext == ".jpg" || file_ext == ".jpeg" || file_ext == ".png") {
		//except jpg/png of other image don't add watermark
		return nil
	}

	img, err := imaging.Open(imgpath)
	if err != nil {
		return err
	}

	watermark, err1 := imaging.Open(watermarkpath)
	if err1 != nil {
		return err1
	}

	log.Debugf("image height : " + strconv.Itoa(img.Bounds().Dy()))

	waterSize := int(math.Floor(float64(img.Bounds().Dy() / 10)))
	waterOffset := waterSize / 10

	log.Debugf("image water offset : %s", strconv.Itoa(waterOffset))

	watermark = imaging.Resize(watermark, waterSize, waterSize, imaging.Lanczos)

	//把水印写到右下角，并向0坐标各偏移10%个像素
	offset := image.Pt(img.Bounds().Dx()-watermark.Bounds().Dx()-waterOffset, img.Bounds().Dy()-watermark.Bounds().Dy()-waterOffset)

	img = imaging.Overlay(img, watermark, offset, 1.0)

	//覆盖图片
	err = imaging.Save(img, imgpath)
	if err != nil {
		log.Infof("watermark failed : %s - %s", err.Error(), filepath.Base(imgpath))
	}

	log.Infof("watermark success - %s", filepath.Base(imgpath))

	return nil
}
