package main

import (
	"fmt"
	"image"
	"math"
	"os"
	"time"

	_ "image/png"

	"github.com/gopxl/pixel/v2"
	"github.com/gopxl/pixel/v2/backends/opengl"
	"golang.org/x/image/colornames"
)

func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return pixel.PictureDataFromImage(img), nil
}

func run() {

	/*TODO

	okay start parsing all this shit big dog
	  Loop

	  Load all the sprite sheets and backgrounds

	  Create entity list manually or from a level init file

	  Create level objects like ground platforms items etc

	  Idea: if things get too slow or N squared divide the level into zones that are ignored unless the player is in an active one

	  Loop{
	    Calculate dt
	    Read player input
	   Update position and behavior and stuff for all e
	    Check for collisions
	    Update any state from that outcome
	    Handle animations

	    Draw everything to the surfaces
	    Draw everything to the screen in the correct order

	    Check level or game finish state to break loop and load next
	*/

	INITIAL_WINDOW_WIDTH := 1024.0
	INITIAL_WINDOW_HEIGHT := 768.0
	BLOCK_SIZE := 10

	fmt.Printf("Booting up with reso: %vx%v, blocksize = %v\n", INITIAL_WINDOW_WIDTH, INITIAL_WINDOW_HEIGHT, BLOCK_SIZE)

	cfg := opengl.WindowConfig{
		Title:  "Bird <shoes>!",
		Bounds: pixel.R(0, 0, INITIAL_WINDOW_WIDTH, INITIAL_WINDOW_HEIGHT),
		VSync:  false,
	}
	win, err := opengl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	spritesheet, err := loadPicture("trees.png")
	if err != nil {
		panic(err)
	}

	batch := pixel.NewBatch(&pixel.TrianglesData{}, spritesheet)

	bird_sheet, err := loadPicture("bird_sheet.png")
	if err != nil {
		panic(err)
	}

	bird_batch := pixel.NewBatch(&pixel.TrianglesData{}, bird_sheet)

	var birdFrames []pixel.Rect
	for x := bird_sheet.Bounds().Min.X; x < bird_sheet.Bounds().Max.X; x += 500 {
		birdFrames = append(birdFrames, pixel.R(x, 0, x+500, 500))
	}

	var (
		camPos       = pixel.ZV
		camSpeed     = 500.0
		camZoom      = 1.0
		camZoomSpeed = 1.2
	)

	var (
		frames = 0
		second = time.Tick(time.Second)
	)

	type guy struct {
		x_pos, y_pos float64
		sprite       pixel.Sprite
	}

	guy_B := guy{
		x_pos:  60,
		y_pos:  100,
		sprite: *pixel.NewSprite(bird_sheet, birdFrames[1]),
	}

	last := time.Now()
	for !win.Closed() {
		dt := time.Since(last).Seconds()
		last = time.Now()

		// batch = pixel.NewBatch(&pixel.TrianglesData{}, spritesheet)

		cam := pixel.IM.Scaled(camPos, camZoom).Moved(win.Bounds().Center().Sub(camPos))
		win.SetMatrix(cam)

		if win.JustPressed(pixel.MouseButtonLeft) {
			// tree := pixel.NewSprite(spritesheet, treesFrames[rand.Intn(len(treesFrames))])
			// mouse := cam.Unproject(win.MousePosition())
			// fmt.Println(mouse)
			// tree.Draw(batch, pixel.IM.Scaled(pixel.ZV, 4).Moved(mouse))
		}
		if win.Pressed(pixel.KeyLeft) {
			camPos.X -= camSpeed * dt
		}
		if win.Pressed(pixel.KeyRight) {
			camPos.X += camSpeed * dt
		}
		if win.Pressed(pixel.KeyDown) {
			camPos.Y -= camSpeed * dt
		}
		if win.Pressed(pixel.KeyUp) {
			camPos.Y += camSpeed * dt
		}
		if win.JustPressed(pixel.KeyQ) {
			camZoom += .1
		}
		if win.JustPressed(pixel.KeyZ) {
			camZoom -= .1
		}
		if win.JustPressed(pixel.KeyEscape) {
			return
		}

		camZoom *= math.Pow(camZoomSpeed, win.MouseScroll().Y)

		win.Clear(colornames.Forestgreen)

		for i := 0.0; i < 50; i++ {
			for j := 0.0; j < 50; j++ {
				// guy_B.sprite.Draw(bird_batch, pixel.IM.Scaled(pixel.ZV, 1).Rotated(pixel.ZV, 0.5*i).Moved(pixel.V(i*500, j*500)))
				guy_B.sprite.Draw(bird_batch, pixel.IM.ScaledXY(pixel.ZV, pixel.V(1, 2)).Rotated(pixel.ZV, 0.5*i).Moved(pixel.V(i*500, j*500)))
			}
		}

		batch.Draw(win)
		bird_batch.Draw(win)
		win.Update()

		batch.Clear()
		bird_batch.Clear()

		frames++
		select {
		case <-second:
			win.SetTitle(fmt.Sprintf("%s | FPS: %d", cfg.Title, frames))
			frames = 0
		default:
		}
	}
}

func main() {
	opengl.Run(run)
}
