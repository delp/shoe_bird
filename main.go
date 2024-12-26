/*TODO ray: marios max velocity in SMB3 is 4.5 pixels per frame
    and his sprite width is 8 pixels
7,0	********
	********
	********
	********
	********
	********
	********
0,0 ********


// Stories
=======
[ ] There is a build system that pushes the latest binary to a place ray can grab it
 * github or my server

[X] There are debug keys that are printed to console that affect physics constants
=======

so if the bird is 500 px, his max velocity should be 250 px per frame

dimensional analysis

pixels per meter: pixels / m
frames per second: frames / sec
pixels per frame: px / frame
*/

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
	"github.com/gopxl/pixel/v2/ext/imdraw"
	"golang.org/x/image/colornames"
)

var (
	INITIAL_WINDOW_WIDTH  = 1024.0
	INITIAL_WINDOW_HEIGHT = 768.0
	BLOCK_SIZE            = 10
	JUMP_IMPULSE          = 2800.0
	GRAVITY               = -9.8 * 500
	RUN_IMPULSE           = 13600.0
	STOP_IMPULSE          = 16650.0

	RUN_TIMER = 5

	JUMP_INCREMENT    = JUMP_IMPULSE * 0.1
	GRAVITY_INCREMENT = GRAVITY * 0.1
	RUN_INCREMENT     = RUN_IMPULSE * 0.1
	STOP_INCREMENT    = STOP_IMPULSE * 0.1
)

const (
	// List of all mouse buttons.
	LEFT Direction = iota
	RIGHT
)

func printConstants() {
	fmt.Println("=========================")
	fmt.Printf("Jump Impulse: %v\n", JUMP_IMPULSE)
	fmt.Printf("Run Impulse: %v\n", RUN_IMPULSE)
	fmt.Printf("Stop Impulse: %v\n", STOP_IMPULSE)
	fmt.Printf("Gravity: %v\n", GRAVITY)
	fmt.Println("=========================")
}

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

type entity interface {
	draw()
	update(dt float64)
}

type Direction int

type guy struct {
	x_pos, y_pos float64
	dx, dy       float64
	maxdx, maxdy float64

	//Animations and sprites
	spritesheet *pixel.Picture
	sprite      pixel.Sprite
	batch       *pixel.Batch
	animations  map[string]pixel.Rect

	//Movement State Information
	runningRight      bool
	runningLeft       bool
	topSpeedRight     bool
	topSpeedLeft      bool
	TurnaroundToLeft  bool
	TurnaroundToRight bool
	Airborne          bool
	Falling           bool

	direction Direction
}

func (g *guy) draw() {
	if g.direction == RIGHT {
		g.sprite.Draw(g.batch, pixel.IM.Moved(pixel.V(g.x_pos, g.y_pos)))
	} else {
		g.sprite.Draw(g.batch, pixel.IM.ScaledXY(pixel.ZV, pixel.V(-1, 1)).Moved(pixel.V(g.x_pos, g.y_pos)))
	}
}

func (g *guy) update(dt float64) {

	if g.dy == 0 {
		g.Airborne = false
		g.sprite = *pixel.NewSprite(*g.spritesheet, g.animations["stand"]) // g.animations["stand"]
	}

	if g.runningRight || g.runningLeft {
		g.sprite = *pixel.NewSprite(*g.spritesheet, g.animations["run1"])
	}

	// if g.dy > 0 {
	// 	g.sprite = *pixel.NewSprite(*g.spritesheet, g.animations["jump1"])
	// }
	// if g.dy < 0 {
	// 	g.sprite = *pixel.NewSprite(*g.spritesheet, g.animations["jump2"])
	// }

	g.dy += GRAVITY * dt

	//Apply stop inertia only if not running OR in turnaround state
	if g.dx > 0 && !g.runningLeft && !g.runningRight {
		g.dx -= STOP_IMPULSE * dt
	} else if g.dx <= 0 && !g.runningLeft && !g.runningRight {
		g.dx += STOP_IMPULSE * dt
	} else if g.dx > 0 && g.runningLeft && !g.Airborne {
		g.dx -= STOP_IMPULSE * dt
	} else if g.dx <= 0 && g.runningRight && !g.Airborne {
		g.dx += STOP_IMPULSE * dt
	}
	g.x_pos += g.dx * dt
	g.y_pos += g.dy * dt

	if g.dx > g.maxdx {
		g.dx = g.maxdx
	}
	if g.dx < -g.maxdx {
		g.dx = -g.maxdx
	}
	if g.dy > g.maxdy {
		g.dy = g.maxdy
	}

}

func run() {

	/*TODO

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

	//The bird is about a meter tall.
	/*
		500 px      9.8 meters   4900 px
		---      *   -----      = ---
		1 meter       s^2           s^2
	*/

	//Initialize the window. Note the VSYNC setting
	cfg := opengl.WindowConfig{
		Title:  "Bird <shoes>!",
		Bounds: pixel.R(0, 0, INITIAL_WINDOW_WIDTH, INITIAL_WINDOW_HEIGHT),
		VSync:  false,
	}
	win, err := opengl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	// Load all the sprite sheets and backgrounds
	bird_sheet, err := loadPicture("bird_sheet.png")
	if err != nil {
		panic(err)
	}

	bird_batch := pixel.NewBatch(&pixel.TrianglesData{}, bird_sheet)

	var birdFrames []pixel.Rect
	for x := bird_sheet.Bounds().Min.X; x < bird_sheet.Bounds().Max.X; x += 500 {
		birdFrames = append(birdFrames, pixel.R(x, 0, x+500, 500))
	}

	//platform drawer
	imd := imdraw.New(bird_sheet)
	imd.Color = colornames.Red
	a := pixel.V(-2000, -200)
	b := pixel.V(50000, -100)
	imd.Push(a, b)
	imd.Rectangle(0)
	//Camera and framerate stuff
	var (
		camPos       = pixel.ZV
		camSpeed     = 1500.0
		camZoom      = .2
		camZoomSpeed = 1.2
	)

	var (
		frames = 0
		second = time.Tick(time.Second)
	)

	// TODO Create entity list manually or from a level init file

	var entities []entity

	bird := guy{
		x_pos:       60,
		y_pos:       100,
		maxdx:       4000,
		maxdy:       6000,
		spritesheet: &bird_sheet,
		sprite:      *pixel.NewSprite(bird_sheet, birdFrames[1]),
		batch:       bird_batch,
		animations:  make(map[string]pixel.Rect),
	}

	bird.animations["jump1"] = pixel.R(0, 0, 500, 500)
	bird.animations["jump2"] = pixel.R(500, 0, 1000, 500)
	bird.animations["stand"] = pixel.R(1000, 0, 1500, 500)
	bird.animations["run1"] = pixel.R(1500, 0, 2000, 500)
	bird.animations["run2"] = pixel.R(2000, 0, 2500, 500)

	entities = append(entities, &bird)

	last := time.Now()

	// TODO Create level objects like ground platforms items etc

	for !win.Closed() {
		dt := time.Since(last).Seconds()
		last = time.Now()

		cam := pixel.IM.Scaled(camPos, camZoom).Moved(win.Bounds().Center().Sub(camPos))
		win.SetMatrix(cam)

		//TODO do you want to do it like this? hack?
		bird.runningRight = false
		bird.runningLeft = false

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
		if win.JustPressed(pixel.KeySpace) {
			bird.Airborne = true
			bird.dy = float64(JUMP_IMPULSE)
		}
		if win.Pressed(pixel.KeyD) {
			if !win.Pressed(pixel.KeyA) {
				bird.dx += float64(RUN_IMPULSE) * dt
				bird.runningRight = true
				bird.direction = RIGHT
			}
		}
		if win.Pressed(pixel.KeyA) {
			if !win.Pressed(pixel.KeyD) {
				//possible states here
				//stopped
				//acel left
				//acel right
				//tops right
				//tops left

				bird.runningLeft = true
				bird.direction = LEFT
				bird.dx -= float64(RUN_IMPULSE) * dt
			}
		}
		if win.JustPressed(pixel.KeyG) {
			fmt.Printf("Run impulse: %v + 100 = %v\n", RUN_IMPULSE, RUN_IMPULSE+100)
			RUN_IMPULSE += 100
		}
		if win.JustPressed(pixel.KeyF) {
			fmt.Printf("Run impulse: %v - 100 = %v\n", RUN_IMPULSE, RUN_IMPULSE-100)
			RUN_IMPULSE -= 100
		}
		if win.Pressed(pixel.KeyT) {
			//Okay, you can just have this "always on" to enable camera tracking, it looks nice, regardless
			//of if I fully understand it lol
			camPos = pixel.Lerp(camPos, pixel.V(bird.x_pos, bird.y_pos), 1-math.Pow(1.0/128, dt))
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

		if win.JustPressed(pixel.KeyU) {
			JUMP_IMPULSE += JUMP_INCREMENT
			printConstants()
		}
		if win.JustPressed(pixel.KeyJ) {
			JUMP_IMPULSE -= JUMP_INCREMENT
			printConstants()
		}
		if win.JustPressed(pixel.KeyI) {
			RUN_IMPULSE += RUN_INCREMENT
			printConstants()
		}
		if win.JustPressed(pixel.KeyK) {
			RUN_IMPULSE -= RUN_INCREMENT
			printConstants()
		}
		if win.JustPressed(pixel.KeyO) {
			STOP_IMPULSE += STOP_INCREMENT
			printConstants()
		}
		if win.JustPressed(pixel.KeyL) {
			STOP_IMPULSE -= STOP_INCREMENT
			printConstants()
		}
		if win.JustPressed(pixel.KeyP) {
			GRAVITY += GRAVITY_INCREMENT
			printConstants()
		}
		if win.JustPressed(pixel.KeySemicolon) {
			GRAVITY -= GRAVITY_INCREMENT
			printConstants()
		}

		camZoom *= math.Pow(camZoomSpeed, win.MouseScroll().Y)

		//Update all entities && collision detection???
		for _, item := range entities {
			item.update(dt)
		}

		//Draw the background
		win.Clear(colornames.Forestgreen)

		//Draw all entities
		for _, item := range entities {
			item.draw()
		}

		//Draw the batch to the window and show everything
		bird_batch.Draw(win)
		imd.Draw(win)
		win.Update()

		//Clear the batch
		bird_batch.Clear()

		if bird.y_pos < 0 {
			bird.y_pos = 0
			bird.dy = 0
		}

		//Framerate calculation stuff
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

	greeting := ` ____  _   _  ___  _____   ____ ___ ____  ____  
/ ___|| | | |/ _ \| ____| | __ )_ _|  _ \|  _ \ 
\___ \| |_| | | | |  _|   |  _ \| || |_) | | | |
 ___) |  _  | |_| | |___  | |_) | ||  _ <| |_| |
|____/|_| |_|\___/|_____| |____/___|_| \_\____/                                   
`
	s := `
JUMP_IMPULSE up:   U  STOP_IMPULSE up:   O
JUMP_IMPULSE down: J  STOP_IMPULSE down: L
RUN_IMPULSE up:    I  GRAVITY up:        P
RUN_IMPULSE down:  K  GRAVITY down:      ;

Use the keys to change the movement physics. They will increment by +/- 10% of the initial value.

`
	fmt.Println(greeting)
	fmt.Printf("Booting up with reso: %vx%v, blocksize = %v\n", INITIAL_WINDOW_WIDTH, INITIAL_WINDOW_HEIGHT, BLOCK_SIZE)
	fmt.Println(s)
	opengl.Run(run)
}
