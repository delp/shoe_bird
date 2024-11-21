package main

import (
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	_ "image/png"

	pixel "github.com/gopxl/pixel/v2"
	"github.com/gopxl/pixel/v2/backends/opengl"
	"github.com/gopxl/pixel/v2/ext/imdraw"
	"github.com/pkg/errors"
	"golang.org/x/image/colornames"
)

const (
	jumpDebugIncrement     = 10
	runspeedDebugIncrement = 10
)

func loadAnimationSheet(sheetPath, descPath string, frameWidth float64) (sheet pixel.Picture, anims map[string][]pixel.Rect, err error) {
	// total hack, nicely format the error at the end, so I don't have to type it every time
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "error loading animation sheet")
		}
	}()

	// open and load the spritesheet
	sheetFile, err := os.Open(sheetPath)
	if err != nil {
		return nil, nil, err
	}
	defer sheetFile.Close()
	sheetImg, _, err := image.Decode(sheetFile)
	if err != nil {
		return nil, nil, err
	}
	sheet = pixel.PictureDataFromImage(sheetImg)

	// create a slice of frames inside the spritesheet
	var frames []pixel.Rect
	for x := 0.0; x+frameWidth <= sheet.Bounds().Max.X; x += frameWidth {
		frames = append(frames, pixel.R(
			x,
			0,
			x+frameWidth,
			sheet.Bounds().H(),
		))
	}

	descFile, err := os.Open(descPath)
	if err != nil {
		return nil, nil, err
	}
	defer descFile.Close()

	anims = make(map[string][]pixel.Rect)

	// load the animation information, name and interval inside the spritesheet
	desc := csv.NewReader(descFile)
	for {
		anim, err := desc.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		name := anim[0]
		start, _ := strconv.Atoi(anim[1])
		end, _ := strconv.Atoi(anim[2])

		anims[name] = frames[start : end+1]
	}

	return sheet, anims, nil
}

type platform struct {
	rect  pixel.Rect
	color color.Color
}

func (p *platform) draw(imd *imdraw.IMDraw) {
	imd.Color = p.color
	imd.Push(p.rect.Min, p.rect.Max)
	imd.Rectangle(0)
}

type birdPhys struct {
	gravity   float64
	runSpeed  float64
	jumpSpeed float64

	rect   pixel.Rect
	hitbox pixel.Rect
	vel    pixel.Vec
	ground bool
}

func (gp *birdPhys) update(dt float64, ctrl pixel.Vec, platforms []platform) {
	// apply controls
	switch {
	case ctrl.X < 0:
		gp.vel.X = -gp.runSpeed
	case ctrl.X > 0:
		gp.vel.X = +gp.runSpeed
	default:
		gp.vel.X = 0
	}

	// apply gravity and velocity
	gp.vel.Y += gp.gravity * dt
	gp.rect = gp.rect.Moved(gp.vel.Scaled(dt))

	// check collisions against each platform
	gp.ground = false
	if gp.vel.Y <= 0 {
		for _, p := range platforms {
			if gp.rect.Max.X <= p.rect.Min.X || gp.rect.Min.X >= p.rect.Max.X {
				continue
			}
			if gp.rect.Min.Y > p.rect.Max.Y || gp.rect.Min.Y < p.rect.Max.Y+gp.vel.Y*dt {
				continue
			}
			gp.vel.Y = 0
			gp.rect = gp.rect.Moved(pixel.V(0, p.rect.Max.Y-gp.rect.Min.Y))
			gp.ground = true
		}
	}

	// jump if on the ground and the player wants to jump
	if gp.ground && ctrl.Y > 0 {
		gp.vel.Y = gp.jumpSpeed
	}
}

type animState int

const (
	idle animState = iota
	running
	jumping
)

type gopherAnim struct {
	sheet pixel.Picture
	anims map[string][]pixel.Rect
	rate  float64

	state       animState
	counter     float64
	jumpCounter int
	dir         float64

	frame pixel.Rect

	sprite *pixel.Sprite
}

func (ga *gopherAnim) update(dt float64, phys *birdPhys) {
	ga.counter += dt

	// determine the new animation state
	var newState animState
	switch {
	case !phys.ground:
		newState = jumping
	case phys.vel.Len() == 0:
		newState = idle
	case phys.vel.Len() > 0:
		newState = running
	}

	// reset the time counter if the state changed
	if ga.state != newState {
		ga.state = newState
		ga.counter = 0
	}

	// determine the correct animation frame
	switch ga.state {
	case idle:
		ga.frame = ga.anims["Front"][0]
	case running:
		i := int(math.Floor(ga.counter / ga.rate))
		ga.frame = ga.anims["Run"][i%len(ga.anims["Run"])]
	case jumping:

		if ga.jumpCounter > 0 {
			ga.frame = ga.anims["Jump"][0]
		} else {
			ga.frame = ga.anims["Jump"][1]
		}
		// speed := phys.vel.Y
		// i := int((-speed/phys.jumpSpeed + 1) / 2 * float64(len(ga.anims["Jump"])))
		// if i < 0 {
		// 	i = 0
		// }
		// if i >= len(ga.anims["Jump"]) {
		// 	i = len(ga.anims["Jump"]) - 1
		// }
		// ga.frame = ga.anims["Jump"][i]
		ga.jumpCounter--
		if ga.jumpCounter < 0 {
			ga.jumpCounter = 0
		}
	}

	// set the facing direction of the gopher
	if phys.vel.X != 0 {
		if phys.vel.X > 0 {
			ga.dir = -1
		} else {
			ga.dir = +1
		}
	}
}

func (ga *gopherAnim) draw(t pixel.Target, phys *birdPhys) {
	if ga.sprite == nil {
		ga.sprite = pixel.NewSprite(nil, pixel.Rect{})
	}
	// draw the correct frame with the correct position and direction
	ga.sprite.Set(ga.sheet, ga.frame)
	ga.sprite.Draw(t, pixel.IM.
		ScaledXY(pixel.ZV, pixel.V(
			phys.rect.W()/ga.sprite.Frame().W(),
			phys.rect.H()/ga.sprite.Frame().H(),
		)).
		ScaledXY(pixel.ZV, pixel.V(-ga.dir, 1)).
		Moved(phys.rect.Center()),
	)
}

type goal struct {
	pos    pixel.Vec
	radius float64
	step   float64

	counter float64
	cols    [5]pixel.RGBA
}

func (g *goal) update(dt float64) {
	g.counter += dt
	for g.counter > g.step {
		g.counter -= g.step
		for i := len(g.cols) - 2; i >= 0; i-- {
			g.cols[i+1] = g.cols[i]
		}
		g.cols[0] = randomNiceColor()
	}
}

func (g *goal) draw(imd *imdraw.IMDraw) {
	for i := len(g.cols) - 1; i >= 0; i-- {
		imd.Color = g.cols[i]
		imd.Push(g.pos)
		imd.Circle(float64(i+1)*g.radius/float64(len(g.cols)), 0)
	}
}

func randomNiceColor() pixel.RGBA {
again:
	r := rand.Float64()
	g := rand.Float64()
	b := rand.Float64()
	len := math.Sqrt(r*r + g*g + b*b)
	if len == 0 {
		goto again
	}
	return pixel.RGB(r/len, g/len, b/len)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func run() {

	birdSheet, birdAnims, err := loadAnimationSheet("bird_sheet.png", "bird_sheet.csv", 500)
	check(err)

	noseSheet, noseAnims, err := loadAnimationSheet("nosey.guy.sheet.png", "nosey.guy.sheet.csv", 235)
	check(err)

	flyguySheet, flyguyAnims, err := loadAnimationSheet("flyguy.sheet.png", "flyguy.sheet.csv", 208)
	check(err)

	cfg := opengl.WindowConfig{
		Title:  "Platformer",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  true,
	}
	win, err := opengl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	phys := &birdPhys{
		gravity:   -8000,
		runSpeed:  4500,
		jumpSpeed: 6000,
		rect:      pixel.R(-500, -500, 500, 500),
		hitbox:    pixel.R(-400, -500, 400, 500),
	}

	flyPhys := &birdPhys{}

	nosePhys := &birdPhys{}

	anim := &gopherAnim{
		sheet: birdSheet,
		anims: birdAnims,
		rate:  1.0 / 10,
		dir:   -1,
	}

	noseAnim := &gopherAnim{
		sheet: noseSheet,
		anims: noseAnims,
		rate:  1.0 / 10,
		dir:   1,
	}

	flyAnim := &gopherAnim{
		sheet: flyguySheet,
		anims: flyguyAnims,
		rate:  1.0 / 10,
		dir:   1,
	}

	// hardcoded level
	platforms := []platform{

		{rect: pixel.R(10000, -300, 12000, -500)},

		{rect: pixel.R(9000, 500, 12000, 400)},
		// {rect: pixel.R(10000, -300, 12000, -500)},
		// {rect: pixel.R(10000, -300, 12000, -500)},
		{rect: pixel.R(-300, -2000, 10000, -2200)},
	}
	for i := 0; i < 1000; i++ {
		platforms = append(platforms, randomPlatform())
	}

	for i := range platforms {
		platforms[i].color = randomNiceColor()
	}

	gol := &goal{
		pos:    pixel.V(-75, 40),
		radius: 750,
		step:   1.0 / 7,
	}

	//canvas := opengl.NewCanvas(pixel.R(-160/2, -120/2, 160/2, 120/2))
	xmax := (1024.0 * 4)
	xmin := -xmax
	ymax := (768.0 * 4)
	ymin := -ymax
	canvas := opengl.NewCanvas(pixel.R(xmin, ymin, xmax, ymax))
	imd := imdraw.New(birdSheet)
	imd.Precision = 32

	camPos := pixel.ZV

	scaleFactor := 6.3
	otherScaleFactor := 6.3

	last := time.Now()
	for !win.Closed() {
		dt := time.Since(last).Seconds()
		fmt.Println(dt)
		last = time.Now()

		// lerp the camera position towards the gopher
		camPos = pixel.Lerp(camPos, phys.rect.Center(), 1-math.Pow(1.0/128, dt))
		cam := pixel.IM.Moved(camPos.Scaled(-1))
		canvas.SetMatrix(cam)

		// slow motion with tab
		if win.Pressed(pixel.KeyTab) {
			dt /= 8
		}

		// restart the level on pressing enter
		if win.JustPressed(pixel.KeyEnter) {

			fmt.Printf("gopher bounds: %v\n", phys.rect.Bounds())
			fmt.Printf("gopher center: %v\n", phys.rect.Center())
			fmt.Printf("camera vector: %v\n", camPos)
			returnVector := phys.rect.Center().Scaled(-1)
			phys.rect = phys.rect.Moved(returnVector)
			phys.vel = pixel.ZV

			fmt.Println("After move: ")
			fmt.Printf("gopher bounds: %v\n", phys.rect.Bounds())
			fmt.Printf("gopher center: %v\n", phys.rect.Center())
			fmt.Printf("camera vector: %v\n", camPos)
		}

		// control the gopher with keys
		ctrl := pixel.ZV
		if win.Pressed(pixel.KeyLeft) {
			ctrl.X--
		}
		if win.Pressed(pixel.KeyRight) {
			ctrl.X++
		}
		if win.JustPressed(pixel.KeyUp) || win.JustPressed(pixel.KeySpace) {
			ctrl.Y = 1
			anim.jumpCounter = 10
		}
		if win.JustPressed(pixel.KeyQ) {
			fmt.Printf("Q: Jumpspeed %v + %v = %v\n", phys.jumpSpeed, jumpDebugIncrement, phys.jumpSpeed+jumpDebugIncrement)
			phys.jumpSpeed += jumpDebugIncrement
		}

		if win.JustPressed(pixel.KeyA) {
			fmt.Printf("A: Jumpspeed %v - %v = %v\n", phys.jumpSpeed, jumpDebugIncrement, phys.jumpSpeed+jumpDebugIncrement)
			phys.jumpSpeed -= jumpDebugIncrement
		}

		if win.JustPressed(pixel.KeyE) {
			fmt.Printf("E: Runspeed %v + %v = %v\n", phys.runSpeed, runspeedDebugIncrement, phys.runSpeed+runspeedDebugIncrement)
			phys.runSpeed += runspeedDebugIncrement
		}
		if win.JustPressed(pixel.KeyW) {
			fmt.Printf("W: Runspeed %v - %v = %v\n", phys.runSpeed, runspeedDebugIncrement, phys.runSpeed-runspeedDebugIncrement)
			phys.runSpeed -= runspeedDebugIncrement
		}
		if win.JustPressed(pixel.KeyZ) {
			fmt.Printf("%v\n", phys.rect.Bounds())
		}
		if win.JustPressed(pixel.KeyF) {
			scaleFactor -= .3
		}
		if win.JustPressed(pixel.KeyG) {
			scaleFactor += .3
		}
		if win.JustPressed(pixel.KeyV) {
			otherScaleFactor -= .3
		}
		if win.JustPressed(pixel.KeyB) {
			otherScaleFactor += .3
		}
		if win.JustPressed(pixel.KeyEscape) {
			return
		}
		if win.JustPressed(pixel.KeyD) {
			fmt.Printf("window bounds: %v\n", win.Bounds())
			fmt.Printf("canvas bounds: %v\n", win.Canvas().Bounds())

		}

		// update the physics and animation
		phys.update(dt, ctrl, platforms)
		gol.update(dt)
		anim.update(dt, phys)
		noseAnim.update(dt, nosePhys)
		flyAnim.update(dt, flyPhys)

		// draw the scene to the canvas using IMDraw
		canvas.Clear(colornames.White)
		imd.Clear()
		for _, p := range platforms {
			p.draw(imd)
		}
		gol.draw(imd)
		anim.draw(imd, phys)
		noseAnim.draw(imd, nosePhys)
		flyAnim.draw(imd, flyPhys)
		imd.Draw(canvas)

		// stretch the canvas to the window
		win.Clear(colornames.White)
		win.SetMatrix(pixel.IM.Scaled(pixel.ZV,
			math.Min(
				win.Bounds().W()/canvas.Bounds().W(),
				win.Bounds().H()/canvas.Bounds().H(),
			),
		).Moved(win.Bounds().Center()))

		//canvas.Draw(win, pixel.IM.Moved(canvas.Bounds().Center()))
		canvas.Draw(win, pixel.IM)
		win.Update()
	}
}

func main() {
	debugInfo :=
		`debug controls:
Q: increase jump  E: increase run
A: decrease jump  W: decrease run`
	fmt.Printf("%v\n", debugInfo)
	opengl.Run(run)
}

func randomPlatform() platform {
	x := rand.Intn(100000)
	y := rand.Intn(100000)

	return platform{
		rect: pixel.R(float64(x), float64(y), float64(x+1500), float64(y-150)),
	}
}
