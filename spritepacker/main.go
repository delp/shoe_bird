package main

import (
	"fmt"
	"image"
	"os"

	"image/png"
	_ "image/png"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: spritepacker <left image> <right image>")
		return
	}
	leftPicpath := os.Args[1]
	rightPicPath := os.Args[2]
	leftPic, err := os.Open(leftPicpath)
	check(err)
	defer leftPic.Close()
	rightPic, err := os.Open(rightPicPath)
	check(err)
	defer rightPic.Close()

	leftImg, _, err := image.Decode(leftPic)
	check(err)

	rightImg, _, err := image.Decode(rightPic)
	check(err)

	w1, h1 := leftImg.Bounds().Dx(), leftImg.Bounds().Dy()
	w2, h2 := rightImg.Bounds().Dx(), rightImg.Bounds().Dy()

	w := w1 + w2
	h := max(h1, h2)

	fmt.Println(w1, h1, w2, h2)

	newImg := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x < w1 {
				newImg.Set(x, y, leftImg.At(x, y))
			} else {
				newImg.Set(x, y, rightImg.At(x-w1, y))
			}
		}
	}

	// Save the combined image to a new file
	f, err := os.Create("combined.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	png.Encode(f, newImg)

}
