package main

import (
	"fmt"
	"image"
	"image/draw"
	_ "image/png"
	"log"
	"math"
	"os"
	"runtime"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
)

const windowWidth = 800
const windowHeight = 600

//camera init
var (
	cameraPos   = mgl32.Vec3([3]float32{0, 0, 3})
	cameraFront = mgl32.Vec3([3]float32{0, 0, -1})
	cameraUp    = mgl32.Vec3([3]float32{0, 1, 0})
	deltaTime   = float64(0.0) // Time between current frame and last frame
	lastFrame   = float64(0.0) // Time of last frame
)

// camera mouse inint
var (
	yaw, pitch   = float32(-90), float32(0) //init cameraFront = {0, 0, -1})
	lastX, lastY float32
	firstMouse   bool
)

func init() {
	// GLFW event handling must run on the main OS thread
	runtime.LockOSThread()
}

func main() {
	if err := glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(windowWidth, windowHeight, "Cube", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	// set mouse call back
	window.SetCursorPosCallback(mouseMoveCallback)
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	// Configure the vertex and fragment shaders
	program, err := newProgram(vertexShader, fragmentShader)
	if err != nil {
		panic(err)
	}

	gl.UseProgram(program)

	projection := mgl32.Perspective(mgl32.DegToRad(45.0), float32(windowWidth)/windowHeight, 0.1, 10.0)
	projectionUniform := gl.GetUniformLocation(program, gl.Str("projection\x00"))
	gl.UniformMatrix4fv(projectionUniform, 1, false, &projection[0])

	camera := mgl32.LookAtV(cameraPos, cameraPos.Add(cameraFront), cameraUp)
	cameraUniform := gl.GetUniformLocation(program, gl.Str("camera\x00"))
	gl.UniformMatrix4fv(cameraUniform, 1, false, &camera[0])

	model := mgl32.Ident4()
	modelUniform := gl.GetUniformLocation(program, gl.Str("model\x00"))
	gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])

	textureUniform := gl.GetUniformLocation(program, gl.Str("tex\x00"))
	gl.Uniform1i(textureUniform, 0)

	gl.BindFragDataLocation(program, 0, gl.Str("outputColor\x00"))

	// Load the texture
	texture, err := newTexture("square.png")
	if err != nil {
		log.Fatalln(err)
	}

	// Configure the vertex data
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, gl.Ptr(cubeVertices), gl.STATIC_DRAW)

	vertAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vert\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointer(vertAttrib, 3, gl.FLOAT, false, 5*4, gl.PtrOffset(0))

	texCoordAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vertTexCoord\x00")))
	gl.EnableVertexAttribArray(texCoordAttrib)
	gl.VertexAttribPointer(texCoordAttrib, 2, gl.FLOAT, false, 5*4, gl.PtrOffset(3*4))

	// Configure global settings
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(1.0, 1.0, 1.0, 1.0)

	angle := 0.0
	previousTime := glfw.GetTime()

	for !window.ShouldClose() {
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

		// Update
		time := glfw.GetTime()
		elapsed := time - previousTime
		previousTime = time

		angle += elapsed

		// Render
		gl.UseProgram(program)
		gl.BindVertexArray(vao)
		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, texture)

		processInput(window)
		// center is z-axi
		camera := mgl32.LookAtV(cameraPos, cameraPos.Add(cameraFront), cameraUp)
		gl.UniformMatrix4fv(cameraUniform, 1, false, &camera[0])

		for i, each := range cubePositions {
			model_t := mgl32.Translate3D(each[0], each[1], each[2])
			model_r := mgl32.HomogRotate3D(float32(i)*20, mgl32.Vec3{0, 1, 0})
			model = model_r.Mul4(model_t)

			gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])
			gl.DrawArrays(gl.TRIANGLES, 0, 6*2*3)
		}

		//make sure to have same speed in different machine
		currentFrame := glfw.GetTime()
		deltaTime = currentFrame - lastFrame
		lastFrame = currentFrame

		// Maintenance
		window.SwapBuffers()
		glfw.PollEvents()
	}
}

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func newTexture(file string) (uint32, error) {
	imgFile, err := os.Open(file)
	if err != nil {
		return 0, fmt.Errorf("texture %q not found on disk: %v", file, err)
	}
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return 0, err
	}

	rgba := image.NewRGBA(img.Bounds())
	if rgba.Stride != rgba.Rect.Size().X*4 {
		return 0, fmt.Errorf("unsupported stride")
	}
	draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Rect.Size().X),
		int32(rgba.Rect.Size().Y),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix))

	return texture, nil
}

var vertexShader = `
#version 410
uniform mat4 projection;
uniform mat4 camera;
uniform mat4 model;
in vec3 vert;
in vec2 vertTexCoord;
out vec2 fragTexCoord;
void main() {
    fragTexCoord = vertTexCoord;
	// gl_Position = projection * camera * model * vec4(vert, 1);
	gl_Position = projection * camera* model * vec4(vert, 1);
}
` + "\x00"

var fragmentShader = `
#version 410
uniform sampler2D tex;
in vec2 fragTexCoord;
out vec4 outputColor;
void main() {
    outputColor = texture(tex, fragTexCoord);
}
` + "\x00"

var cubeVertices = []float32{
	// Bottom
	-0.5, -0.5, -0.5, 0.0, 0.0,
	0.5, -0.5, -0.5, 0.5, 0.0,
	-0.5, -0.5, 0.5, 0.0, 0.5,
	0.5, -0.5, -0.5, 0.5, 0.0,
	0.5, -0.5, 0.5, 0.5, 0.5,
	-0.5, -0.5, 0.5, 0.0, 0.5,

	// Top
	-0.5, 0.5, -0.5, 0.0, 0.0,
	-0.5, 0.5, 0.5, 0.0, 0.5,
	0.5, 0.5, -0.5, 0.5, 0.0,
	0.5, 0.5, -0.5, 0.5, 0.0,
	-0.5, 0.5, 0.5, 0.0, 0.5,
	0.5, 0.5, 0.5, 0.5, 0.5,

	// Front
	-0.5, -0.5, 0.5, 0.5, 0.0,
	0.5, -0.5, 0.5, 0.0, 0.0,
	-0.5, 0.5, 0.5, 0.5, 0.5,
	0.5, -0.5, 0.5, 0.0, 0.0,
	0.5, 0.5, 0.5, 0.0, 0.5,
	-0.5, 0.5, 0.5, 0.5, 0.5,

	// Back
	-0.5, -0.5, -0.5, 0.0, 0.0,
	-0.5, 0.5, -0.5, 0.0, 0.5,
	0.5, -0.5, -0.5, 0.5, 0.0,
	0.5, -0.5, -0.5, 0.5, 0.0,
	-0.5, 0.5, -0.5, 0.0, 0.5,
	0.5, 0.5, -0.5, 0.5, 0.5,

	// Left
	-0.5, -0.5, 0.5, 0.0, 0.5,
	-0.5, 0.5, -0.5, 0.5, 0.0,
	-0.5, -0.5, -0.5, 0.0, 0.0,
	-0.5, -0.5, 0.5, 0.0, 0.5,
	-0.5, 0.5, 0.5, 0.5, 0.5,
	-0.5, 0.5, -0.5, 0.5, 0.0,

	// Right
	0.5, -0.5, 0.5, 0.5, 0.5,
	0.5, -0.5, -0.5, 0.5, 0.0,
	0.5, 0.5, -0.5, 0.0, 0.0,
	0.5, -0.5, 0.5, 0.5, 0.5,
	0.5, 0.5, -0.5, 0.0, 0.0,
	0.5, 0.5, 0.5, 0.0, 0.5,
}

var cubePositions = [][]float32{
	[]float32{0.0, 0.0, 0.0},
	[]float32{2.0, 5.0, -15.0},
	[]float32{-1.5, -2.2, -2.5},
	[]float32{-3.8, -2.0, -12.},
	[]float32{2.4, -0.4, -3.5},
	[]float32{-1.7, 3.0, -7.5},
	[]float32{1.3, -2.0, -2.5},
	[]float32{1.5, 2.0, -2.5},
	[]float32{1.5, 0.2, -1.5},
	[]float32{-1.3, 1.0, -1.5},
}

func processInput(win *glfw.Window) {
	cameraSpeed := float32(2.5 * deltaTime)
	if win.GetKey(glfw.KeyW) == glfw.Press {
		cameraPos = cameraPos.Add(cameraFront.Mul(cameraSpeed))
	}
	if win.GetKey(glfw.KeyS) == glfw.Press {
		cameraPos = cameraPos.Sub(cameraFront.Mul(cameraSpeed))
	}
	if win.GetKey(glfw.KeyA) == glfw.Press {
		cameraRight := cameraFront.Cross(cameraUp).Normalize()
		cameraPos = cameraPos.Add(cameraRight.Mul(cameraSpeed))
	}
	if win.GetKey(glfw.KeyD) == glfw.Press {
		cameraRight := cameraFront.Cross(cameraUp).Normalize()
		cameraPos = cameraPos.Sub(cameraRight.Mul(cameraSpeed))
	}
}

//refer to https://learnopengl.com/Getting-started/Camera
func mouseMoveCallback(w *glfw.Window, xpos float64, ypos float64) {
	//only handle left mouse button
	sensitivity := float32(0.1)
	if w.GetMouseButton(glfw.MouseButtonLeft) == glfw.Press {
		if firstMouse {
			lastX = float32(xpos)
			lastY = float32(ypos)
			firstMouse = false
		}
		xoffset := float32(xpos) - lastX
		yoffset := lastY - float32(ypos)
		lastX = float32(xpos)
		lastY = float32(ypos)

		xoffset *= sensitivity
		yoffset *= sensitivity

		yaw += xoffset
		pitch += yoffset

		// if pitch > 89.0 {
		// 	pitch = 89.0
		// }
		// if pitch < -89.0 {
		// 	pitch = -89.0
		// }

		x := math.Cos(float64(mgl32.DegToRad(yaw))) * math.Cos(float64(mgl32.DegToRad(pitch)))
		y := math.Sin(float64(mgl32.DegToRad(pitch)))
		z := math.Sin(float64(mgl32.DegToRad(yaw))) * math.Cos(float64(mgl32.DegToRad(pitch)))
		cameraFront = mgl32.Vec3([3]float32{float32(x), float32(y), float32(z)}).Normalize()
		// fmt.Println(x, y, z)
	} else if w.GetMouseButton(glfw.MouseButtonLeft) == glfw.Release {
		firstMouse = true
	}

}
